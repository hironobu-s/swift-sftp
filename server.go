package main

import (
	"bufio"
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

const (
	PasswordSalt = "swift-sftp"
)

func StartServer(conf Config) error {
	// Prepare server config and client
	sConf, client, err := initServer(conf)
	if err != nil {
		return err
	}

	// swift
	swift := NewSwift(conf)
	if err = swift.Init(); err != nil {
		return err
	}
	log.Infof("Use container '%s'", conf.Container)

	// Start server
	listener, err := net.Listen("tcp", conf.BindAddress)
	if err != nil {
		return err
	}
	log.Printf("Listen: %s", conf.BindAddress)

	for {
		nConn, err := listener.Accept()
		if err != nil {
			return err
		}

		log.Printf("Connect from %s", nConn.RemoteAddr())
		go func() {
			err := handleClient(conf, sConf, swift, nConn, client)
			if err == nil {
				return
			}

			serr, ok := err.(*ssh.ServerAuthError)
			if !ok {
				log.Warnf("Client error: %v", err)
				return
			}

			for _, err = range serr.Errors {
				log.Warnf("Client Error: %v", err)
			}
		}()
	}
}

// Generate ECDSA private key
func generatePrivateKey(path string) error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	encoded, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}

	data := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: encoded})

	return ioutil.WriteFile(path, data, 0600)
}

func initServer(conf Config) (sConf *ssh.ServerConfig, client *Client, err error) {
	client = &Client{}

	// generate host key if it not exists.
	if _, err = os.Stat(conf.HostPrivateKeyPath); err != nil {
		if err = generatePrivateKey(conf.HostPrivateKeyPath); err != nil {
			return nil, client, err
		}
	}

	sConf = &ssh.ServerConfig{
		PublicKeyCallback: authPkey(conf, client),
	}

	// Add password authentication method if password file exists
	s, err := os.Stat(conf.PasswordFilePath)
	if !s.IsDir() && err == nil {
		sConf.PasswordCallback = authPassword(conf, client)
	}

	// host private key
	pkeyBytes, err := ioutil.ReadFile(conf.HostPrivateKeyPath)
	if err != nil {
		return nil, client, err
	}

	pkey, err := ssh.ParsePrivateKey(pkeyBytes)
	if err != nil {
		return nil, client, err
	}
	sConf.AddHostKey(pkey)

	return sConf, client, nil
}

func authPkey(conf Config, client *Client) func(c ssh.ConnMetadata, pkey ssh.PublicKey) (*ssh.Permissions, error) {
	return func(c ssh.ConnMetadata, pkey ssh.PublicKey) (*ssh.Permissions, error) {
		authorizedKeysBytes, err := ioutil.ReadFile(conf.AuthorizedKeysPath)
		if err != nil {
			return nil, err
		}

		authorizedKeysMap := map[string]bool{}
		for len(authorizedKeysBytes) > 0 {
			pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
			if err != nil {
				return nil, err
			}

			authorizedKeysMap[string(pubKey.Marshal())] = true
			authorizedKeysBytes = rest
		}

		if authorizedKeysMap[string(pkey.Marshal())] {
			client.SessionID = fmt.Sprintf("%x", c.SessionID())
			client.Username = c.User()
			client.RemoteAddr = c.RemoteAddr().String()

			return &ssh.Permissions{
				// Record the public key used for authentication.
				Extensions: map[string]string{
					"pubkey-fp": ssh.FingerprintSHA256(pkey),
				},
			}, nil
		}
		return nil, fmt.Errorf("unknown public key for %q", c.User())
	}
}

func GenerateHashedPassword(username string, plainPassword []byte) (hashed []byte) {
	buf := bytes.NewBuffer(make([]byte, len(username)+len(plainPassword)+len(PasswordSalt)))
	buf.WriteString(username)
	buf.Write(plainPassword)
	buf.WriteString(PasswordSalt)

	b := sha256.Sum256(buf.Bytes())
	hashed = make([]byte, 64)
	hex.Encode(hashed, b[:])
	return hashed
}

func authPassword(conf Config, client *Client) func(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {

	return func(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		f, err := os.Open(conf.PasswordFilePath)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		r := bufio.NewReader(f)

		for {
			line, _, err := r.ReadLine()
			if err == io.EOF {
				break
			} else if err != nil {
				return nil, err
			}

			var listUser []byte
			var listPass []byte
			for i := 0; i < len(line); i++ {
				if line[i] == ':' {
					listUser = line[:i]
					listPass = line[i+1:]
					break
				}
			}

			hashed := GenerateHashedPassword(c.User(), password)
			if subtle.ConstantTimeCompare(listUser, []byte(c.User())) == 1 &&
				subtle.ConstantTimeCompare(listPass, hashed) == 1 {
				// authorized
				client.SessionID = fmt.Sprintf("%x", c.SessionID())
				client.Username = c.User()
				client.RemoteAddr = c.RemoteAddr().String()
				return nil, nil
			}
		}

		return nil, fmt.Errorf("password rejected for %q", c.User())
	}
}

func handleClient(conf Config, sConf *ssh.ServerConfig, swift *Swift, nConn net.Conn, client *Client) error {
	_, chans, reqs, err := ssh.NewServerConn(nConn, sConf)
	if err != nil {
		return err
	}

	// add some fields based on the client to logger
	log = log.WithFields(logrus.Fields{
		"client": client,
	})
	log.Infof("Session opened for %s@%s", client.Username, client.RemoteAddr)

	go ssh.DiscardRequests(reqs)

	for nchan := range chans {
		if nchan.ChannelType() != "session" {
			msg := fmt.Sprintf("The request was rejected because of unknown channel type. [%s]", nchan.ChannelType())
			log.Warn(msg)
			nchan.Reject(ssh.UnknownChannelType, msg)
			continue
		}
		log.Debugf("Channel is accepted[type=%s]", nchan.ChannelType())

		channel, requests, err := nchan.Accept()
		if err != nil {
			return err
		}

		go func(in <-chan *ssh.Request) {
			for req := range in {
				log.Debugf("Handling request [type=%s]", req.Type)

				// We only handle the request that has type of "subsystem".
				ok := false
				if req.Type == "subsystem" && string(req.Payload[4:]) == "sftp" {
					ok = true
				}
				req.Reply(ok, nil)
			}
		}(requests)

		// sftp
		if err = StartSftpSession(swift, channel); err != nil {
			return err
		}
	}

	log.Infof("Session closed for %s@%s", client.Username, client.RemoteAddr)

	return nil
}

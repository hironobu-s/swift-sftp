package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"crypto/x509"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

func StartServer(conf Config) error {
	// Prepare server config
	sConf, err := initServer(conf)
	if err != nil {
		return err
	}

	// swift
	swift := NewSwift(conf)
	if err = swift.Init(); err != nil {
		return err
	}

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

		log.Printf("Accepted client from %s", nConn.RemoteAddr())
		go func() {
			err := handleClient(conf, sConf, swift, nConn)
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

func initServer(conf Config) (sConf *ssh.ServerConfig, err error) {
	// generate host key if it not exists.
	if _, err = os.Stat(conf.HostPrivateKeyPath); err != nil {
		if err = generatePrivateKey(conf.HostPrivateKeyPath); err != nil {
			return nil, err
		}
	}

	authPkey := func(c ssh.ConnMetadata, pkey ssh.PublicKey) (*ssh.Permissions, error) {
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
			return &ssh.Permissions{
				// Record the public key used for authentication.
				Extensions: map[string]string{
					"pubkey-fp": ssh.FingerprintSHA256(pkey),
				},
			}, nil
		}
		return nil, fmt.Errorf("unknown public key for %q", c.User())
	}

	authPassword := func(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		if c.User() == "hiro" && string(password) == "test123" {
			return nil, nil
		}
		return nil, fmt.Errorf("password rejected for %q", c.User())
	}

	sConf = &ssh.ServerConfig{
		PasswordCallback:  authPassword,
		PublicKeyCallback: authPkey,
	}

	// host private key
	pkeyBytes, err := ioutil.ReadFile(conf.HostPrivateKeyPath)
	if err != nil {
		return nil, err
	}

	pkey, err := ssh.ParsePrivateKey(pkeyBytes)
	if err != nil {
		return nil, err
	}
	sConf.AddHostKey(pkey)

	return sConf, nil
}

func handleClient(conf Config, sConf *ssh.ServerConfig, swift *Swift, nConn net.Conn) error {
	_, chans, reqs, err := ssh.NewServerConn(nConn, sConf)
	if err != nil {
		return err
	}

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
	return nil
}

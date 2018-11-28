package main

import (
	"fmt"
	"io/ioutil"
	"net"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

func StartServer(conf Config) error {
	// Prepare server config
	sConf, err := serverConfig(conf)
	if err != nil {
		return err
	}

	// swift
	swift := NewSwift(conf)
	if err = swift.Init(); err != nil {
		return err
	}

	// Start server
	listenAddr := fmt.Sprintf("%s:%d", conf.ListenAddress, conf.ListenPort)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	log.Debugf("Listening for %s", listenAddr)

	for {
		nConn, err := listener.Accept()
		if err != nil {
			return err
		}

		log.Infof("Accept a client from %s", nConn.RemoteAddr())
		go func() {
			err := handleClient(conf, sConf, swift, nConn)
			if err != nil {
				log.Warnf("Client error: %v", err)
			}
		}()
	}
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

func serverConfig(conf Config) (sConf *ssh.ServerConfig, err error) {
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

	sConf = &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},

		PublicKeyCallback: func(c ssh.ConnMetadata, pkey ssh.PublicKey) (*ssh.Permissions, error) {
			if authorizedKeysMap[string(pkey.Marshal())] {
				return &ssh.Permissions{
					// Record the public key used for authentication.
					Extensions: map[string]string{
						"pubkey-fp": ssh.FingerprintSHA256(pkey),
					},
				}, nil
			}
			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
	}

	// private key of server
	pkeyBytes, err := ioutil.ReadFile(conf.ServerPrivateKeyPath)
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

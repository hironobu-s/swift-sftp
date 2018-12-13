package main

import (
	"io"

	"github.com/pkg/sftp"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

func StartSftpSession(swift *Swift, channel ssh.Channel, client *Client) (err error) {
	// logger with client
	clog := log.WithFields(logrus.Fields{
		"client": client,
	})

	clog.Debug("Starting SFTP session.")

	fs := NewSwiftFS(swift)
	fs.SetLogger(clog)
	handler := sftp.Handlers{fs, fs, fs, fs}

	server := sftp.NewRequestServer(channel, handler)

	log.Debug("Initialized sftp server")

	if err = server.Serve(); err == io.EOF {
		log.Debug("End sftp session")

		return server.Close()

	} else if err != nil {
		return err
	}
	return nil
}

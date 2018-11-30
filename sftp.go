package main

import (
	"io"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func StartSftpSession(swift *Swift, channel ssh.Channel) (err error) {
	log.Debug("Starting SFTP session.")

	fs := NewSwiftFS(swift)
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

package main

import (
	"io"
	"os"

	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

func StartSftpSession(channel ssh.Channel) error {
	log.Info("Start sftp session")
	sftpOptions := []sftp.ServerOption{
		sftp.WithDebug(os.Stdout),
		sftp.ReadOnly(),
	}

	server, err := sftp.NewServer(channel, sftpOptions...)
	if err != nil {
		return err
	}

	if err = server.Serve(); err == io.EOF {
		log.Info("End sftp session")
		return nil

	} else if err != nil {
		return err
	}
	return nil
}

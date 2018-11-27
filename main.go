package main

import (
	"os"

	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "conoha transfer for SFTP"
	app.Commands = []cli.Command{
		cli.Command{
			Name:      "server",
			ShortName: "s",
			Usage:     "Start sftp server",
			Action:    server,
		},
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func server(c *cli.Context) (err error) {
	enableDebugTransport()

	log.SetLevel(log.DebugLevel)
	log.Info("Starting server...")

	conf := Config{
		ListenAddress: "127.0.0.1",
		ListenPort:    10022,
		Container:     "test",
	}

	if conf.ServerPrivateKeyPath, err = filepath.Abs("./misc/server.key"); err != nil {
		return err
	}

	if conf.AuthorizedKeysPath, err = filepath.Abs("./misc/authorized_keys"); err != nil {
		return err
	}

	return StartServer(conf)
}

type Config struct {
	// generic options
	CreateContainerIfNotExists bool

	// network parameters
	ListenAddress string
	ListenPort    int

	// ssh keys
	ServerPrivateKeyPath string
	AuthorizedKeysPath   string

	// Required parameters for OpenStack
	Container string
	Region    string

	// Optional parameters for OpenStack
	// If those are not given, We use environment variables like OS_USERNAME to authenticate the client.
	IdentityEndpoint string
	UserID           string
	Username         string
	Password         string
	DomainID         string
	DomainName       string
	TenantID         string
	TenantName       string
}

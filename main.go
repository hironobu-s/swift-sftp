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
	log.SetLevel(log.DebugLevel)
	log.Info("Starting server...")

	conf := Config{
		ListenAddress: "127.0.0.1",
		ListenPort:    10022,
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
	ListenAddress string
	ListenPort    int

	ServerPrivateKeyPath string
	AuthorizedKeysPath   string
}

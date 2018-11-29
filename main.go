package main

import (
	"io"
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
		cli.Command{
			Name:   "test",
			Usage:  "test run",
			Action: test,
		},
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func test(c *cli.Context) (err error) {
	enableDebugTransport()
	log.SetLevel(log.DebugLevel)

	conf := Config{
		ListenAddress: "127.0.0.1",
		ListenPort:    10022,
		Container:     "test",
	}
	s := NewSwift(conf)
	if err = s.Init(); err != nil {
		return err
	}

	log.Debugf("Start downloading")
	rs, size, err := s.Download("go1.10.2.linux-amd64.tar.gz")
	log.Debugf("downloading...")
	if err != nil {
		return err
	}

	log.Debugf("create tmpfile...")
	f, err := os.OpenFile("tmp.dat", os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	log.Debugf("copying...")
	io.Copy(f, rs)

	_ = rs
	_ = size
	log.Debugf("End downloading")

	return nil
}

func server(c *cli.Context) (err error) {
	enableDebugTransport()

	log.SetLevel(log.DebugLevel)
	log.Debugf("Starting server...")

	conf := Config{
		ListenAddress: "0.0.0.0",
		//ListenAddress: "127.0.0.1",
		ListenPort: 10022,
		Container:  "test",
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

package main

import (
	"fmt"
	"io"
	"os"

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
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "container,c",
					Usage: "Specify container name",
				},
				cli.StringFlag{
					Name:  "source-address,a",
					Usage: "Source address of connection",
					Value: "127.0.0.1",
				},
				cli.IntFlag{
					Name:  "port,p",
					Usage: "Port to listen",
					Value: 10022,
				},
				cli.StringFlag{
					Name:  "authorized-keys,k",
					Usage: "File path of authorized_keys",
					Value: "~/.ssh/authorized_keys2",
				},
			},
			Action: server,
		},
		cli.Command{
			Name:   "test",
			Usage:  "test run",
			Action: test,
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		os.Exit(1)
	}
}

func test(c *cli.Context) (err error) {
	enableDebugTransport()
	log.SetLevel(log.DebugLevel)

	conf := Config{
		BindAddress: "127.0.0.1:10022",
		Container:   "test",
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
	// enableDebugTransport()

	// log.SetLevel(log.DebugLevel)
	log.Printf("Starting SFTP server...")

	conf := Config{}
	if err = conf.Init(c); err != nil {
		return err
	}
	conf.Container = c.String("container")

	return StartServer(conf)
}

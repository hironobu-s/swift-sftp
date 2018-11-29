package main

import (
	"fmt"
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
				cli.BoolFlag{
					Name:  "debug,d",
					Usage: "Enable debug output",
				},
				cli.StringFlag{
					Name:  "container,c",
					Usage: "Container name",
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
					Value: "~/.ssh/authorized_keys",
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
	log.SetFormatter(&OriginalFormatter{})

	return nil
}

func server(c *cli.Context) (err error) {
	if c.Bool("debug") {
		enableDebugTransport()
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetFormatter(&OriginalFormatter{})
	}

	conf := Config{}
	if err = conf.Init(c); err != nil {
		return err
	}
	conf.Container = c.String("container")

	log.Infof("Starting SFTP server...")

	return StartServer(conf)
}

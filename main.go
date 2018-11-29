package main

import (
	"fmt"
	"os"
	"time"

	"bytes"

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

type OriginalFormatter struct {
}

func (f *OriginalFormatter) Format(e *log.Entry) ([]byte, error) {
	t := time.Now()
	data := bytes.NewBuffer(make([]byte, 0, 128))
	for k, v := range e.Data {
		data.WriteString(fmt.Sprintf("%s=%s", k, v))
	}

	var msg string
	if data.Len() > 0 {
		msg = fmt.Sprintf("[%s] %s (%s)\n", t.Format("2006-01-02 15:04:05"), e.Message, data)
	} else {
		msg = fmt.Sprintf("[%s] %s\n", t.Format("2006-01-02 15:04:05"), e.Message)
	}
	return []byte(msg), nil
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

	log.Printf("Starting SFTP server...")

	return StartServer(conf)
}

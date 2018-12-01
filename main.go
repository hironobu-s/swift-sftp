package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh/terminal"
)

var log *logrus.Entry

func main() {
	app := cli.NewApp()
	app.Name = "swift-sftp"
	app.Commands = []cli.Command{
		cli.Command{
			Name:      "server",
			ShortName: "s",
			Usage:     "Start SFTP server",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "debug,d",
					Usage: "Enable debug output",
				},
				cli.StringFlag{
					Name:  "address,a",
					Usage: "Source address of connection",
					Value: "localhost:10022",
				},
				cli.StringFlag{
					Name:  "password-file",
					Usage: "Path of password-file. If provided, password authentication is enabled",
					Value: "",
				},
				cli.StringFlag{
					Name:  "authorized-keys,k",
					Usage: "Path of authorized_keys file",
					Value: "~/.ssh/authorized_keys",
				},
			},
			ArgsUsage: "<container>",
			Action:    server,
		},

		cli.Command{
			Name:      "gen-password",
			ShortName: "p",
			Usage:     "Generate password",
			Action:    genPassword,
			ArgsUsage: "[username]",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "format,f",
					Usage: "Output in password-file format. (If not provided, print only password)",
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		os.Exit(1)
	}
}

func server(c *cli.Context) (err error) {
	// log
	l := logrus.New()
	if c.Bool("debug") {
		enableDebugTransport()
		l.SetLevel(logrus.DebugLevel)
	} else {
		l.SetFormatter(&SftpLogFormatter{})
	}
	log = logrus.NewEntry(l)

	opts := ConfigInitOpts{}
	opts.FromContext(c)

	conf := Config{}
	if err = conf.Init(opts); err != nil {
		return err
	}

	log.Infof("Starting SFTP server")

	return StartServer(conf)
}

func genPassword(c *cli.Context) (err error) {
	if c.NArg() != 1 {
		return errors.New("Parameter 'username' required")
	}
	username := c.Args()[0]

	fmt.Fprintf(os.Stderr, "Password: ")
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return err
	} else if len(password) == 0 {
		return nil
	}
	fmt.Println()

	hashed := GenerateHashedPassword(username, password)
	if c.Bool("format") {
		fmt.Fprintf(os.Stdout, "%s:%s", username, hashed)
	} else {
		fmt.Fprintf(os.Stdout, "%s", hashed)
	}
	fmt.Println()
	return nil
}

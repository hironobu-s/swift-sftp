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

var (
	log     *logrus.Entry
	version string
)

func main() {
	app := cli.NewApp()
	app.Name = "swift-sftp"
	app.Author = "Hironobu Saito"
	app.Usage = "SFTP server for OpenStack Swift"

	app.Version = version
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
					Name:  "container,c",
					Usage: "Set container name",
					Value: "",
				},
				cli.BoolFlag{
					Name:  "create-container",
					Usage: "Create container if not exist",
				},
				cli.StringFlag{
					Name:  "config-file,f",
					Usage: "Set configuration file",
					Value: "",
				},
				cli.StringFlag{
					Name:  "address,a",
					Usage: "Set bind address of connection",
					Value: "127.0.0.1:20022",
				},
				cli.StringFlag{
					Name:  "password-file,p",
					Usage: "Set password-file.",
					Value: "",
				},
				cli.StringFlag{
					Name:  "server-key,s",
					Usage: "Set server key file",
					Value: "./server.key",
				},
				cli.StringFlag{
					Name:  "authorized-keys,k",
					Usage: "Set authorized_keys file",
					Value: "~/.ssh/authorized_keys",
				},
				cli.IntFlag{
					Name:  "swift-timeout",
					Usage: "Set timeout for Swift (sec).",
					Value: 180,
				},
			},

			HideHelp: true,
			Action:   server,
		},

		cli.Command{
			Name:      "gen-password-hash",
			ShortName: "g",
			Usage:     "Generate password hash",
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

func server(ctx *cli.Context) (err error) {
	// log
	l := logrus.New()
	if ctx.Bool("debug") {
		enableDebugTransport()
		l.SetLevel(logrus.DebugLevel)
	} else {
		l.SetFormatter(&SftpLogFormatter{})
	}
	log = logrus.NewEntry(l)

	// intialize configuration
	c := Config{}
	if ctx.String("config-file") != "" {
		if err = c.LoadFromFile(ctx.String("config-file")); err != nil {
			return err
		}
	} else {
		if err = c.LoadFromContext(ctx); err != nil {
			return err
		}
	}

	if err = c.Init(); err != nil {
		return err
	}

	log.Infof("Starting SFTP server")

	return StartServer(c)
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

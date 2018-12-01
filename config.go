package main

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/urfave/cli"
)

type Config struct {
	// It's ~/.swift-sftp
	ConfigDir string

	// container creation
	CreateContainerIfNotExists bool

	// password file for password authentication
	PasswordFilePath string

	// network parameters
	BindAddress string

	// ssh keys
	HostPrivateKeyPath string
	AuthorizedKeysPath string

	// Container name
	Container string

	// Optional parameters for OpenStack
	// If those are not given, We use environment variables like OS_USERNAME to authenticate the client.
	OsIdentityEndpoint string
	OsUserID           string
	OsUsername         string
	OsPassword         string
	OsDomainID         string
	OsDomainName       string
	OsTenantID         string
	OsTenantName       string
	OsRegion           string
}

type ConfigInitOpts struct {
	Container          string
	Address            string
	Port               int
	PasswordFilePath   string
	AuthorizedKeysPath string
}

func (c *ConfigInitOpts) FromContext(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		c.Container = ctx.Args()[0]
	}
	c.Address = ctx.String("address")
	c.PasswordFilePath = ctx.String("password-file")
	c.AuthorizedKeysPath = ctx.String("authorized-keys")
}

func (c *Config) Init(opts ConfigInitOpts) (err error) {
	// temporary directory
	u, err := user.Current()
	if err != nil {
		return err
	}
	dir := filepath.Join(u.HomeDir, ".swift-sftp")
	if _, err = os.Stat(dir); err != nil {
		if err = os.Mkdir(dir, 0700); err != nil {
			return err
		}
	}
	c.ConfigDir = dir

	// container
	c.Container = opts.Container
	if c.Container == "" {
		return errors.New("Parameter 'container' required")
	}

	// default values
	c.BindAddress = opts.Address
	c.HostPrivateKeyPath = filepath.Join(c.ConfigDir, "server.key")

	// resolve the path including "~" manually
	var path string

	if opts.PasswordFilePath != "" {
		path = strings.Replace(opts.PasswordFilePath, "~", u.HomeDir, 1)
		path, err = filepath.Abs(path)
		if err != nil {
			return err
		}
		if _, err = os.Stat(path); err != nil {
			return fmt.Errorf("Password file '%s' is not found", opts.PasswordFilePath)
		}
		c.PasswordFilePath = path
	}

	if opts.AuthorizedKeysPath != "" {
		path = strings.Replace(opts.AuthorizedKeysPath, "~", u.HomeDir, 1)
		path, err = filepath.Abs(path)
		if err != nil {
			return err
		}
		if _, err = os.Stat(path); err != nil {
			return fmt.Errorf("Authorized keys file '%s' is not found", opts.AuthorizedKeysPath)
		}
		c.AuthorizedKeysPath = path

	} else {
		return fmt.Errorf("Authorized keys file is required")
	}

	return nil
}

package main

import (
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

func (c *Config) Init(ctx *cli.Context) (err error) {
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
	c.Container = ctx.String("container")

	// default values
	c.BindAddress = fmt.Sprintf("%s:%d", ctx.String("source-address"), ctx.Int("port"))
	c.HostPrivateKeyPath = filepath.Join(c.ConfigDir, "server.key")

	// resolve the path including "~" manually
	var path string

	if ctx.String("password-file") != "" {
		path = strings.Replace(ctx.String("password-file"), "~", u.HomeDir, 1)
		path, err = filepath.Abs(path)
		if err != nil {
			return err
		}
		if _, err = os.Stat(path); err != nil {
			return fmt.Errorf("Password file '%s' is not found", ctx.String("password-file"))
		}
		c.PasswordFilePath = path
	}

	if ctx.String("authorized-keys") != "" {
		path = strings.Replace(ctx.String("authorized-keys"), "~", u.HomeDir, 1)
		path, err = filepath.Abs(path)
		if err != nil {
			return err
		}
		if _, err = os.Stat(path); err != nil {
			return fmt.Errorf("Authorized keys file '%s' is not found", ctx.String("authorized-keys"))
		}
		c.AuthorizedKeysPath = path

	} else {
		return fmt.Errorf("Authorized keys file is required")
	}

	return nil
}

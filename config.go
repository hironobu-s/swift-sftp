package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/urfave/cli"
)

type Config struct {
	ConfigDir                  string
	CreateContainerIfNotExists bool

	// network parameters
	BindAddress string

	// ssh keys
	HostPrivateKeyPath string
	AuthorizedKeysPath string

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
	c.AuthorizedKeysPath = ctx.String("authorized-keys")

	return nil
}

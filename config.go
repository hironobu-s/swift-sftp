package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"os/user"

	"github.com/BurntSushi/toml"
	"github.com/urfave/cli"
)

type Config struct {
	// container creation
	CreateContainerIfNotExists bool `toml:"create_container"`

	// password file for password authentication
	PasswordFilePath string `toml:"password_file"`

	// network parameters
	BindAddress string `toml:"bind_address"`

	// ssh keys
	ServerKeyPath      string `toml:"server_key"`
	AuthorizedKeysPath string `toml:"authorized_keys"`

	// Container name
	Container string `toml:"container"`

	// Timeout for downloading and uploading (sec)
	SwiftTimeout int `toml:"swift_timeout"`

	// Optional parameters for OpenStack
	// If those are not given, We use environment variables like OS_USERNAME to authenticate the client.
	OsIdentityEndpoint string `toml:"os_identity_endpoint"`
	OsUserID           string `toml:"os_user_id"`
	OsUsername         string `toml:"os_username"`
	OsPassword         string `toml:"os_password"`
	OsDomainID         string `toml:"os_domain_id"`
	OsDomainName       string `toml:"os_domain_name"`
	OsTenantID         string `toml:"os_tenant_id"`
	OsTenantName       string `toml:"os_tenant_name"`
	OsRegion           string `toml:"os_region"`
}

func (c *Config) LoadFromContext(ctx *cli.Context) error {
	c.BindAddress = ctx.String("address")
	c.Container = ctx.String("container")
	c.PasswordFilePath = ctx.String("password-file")
	c.ServerKeyPath = ctx.String("server-key")
	c.AuthorizedKeysPath = ctx.String("authorized-keys")
	c.CreateContainerIfNotExists = ctx.Bool("create-container")
	c.SwiftTimeout = ctx.Int("swift-timeout")

	return nil
}

func (c *Config) LoadFromFile(filename string) error {
	if _, err := os.Stat(filename); err != nil {
		return fmt.Errorf("Config file '%s' is not found", filename)
	}

	_, err := toml.DecodeFile(filename, &c)
	return err
}

func (c *Config) Init() (err error) {
	// container
	if c.Container == "" {
		return errors.New("Parameter 'container' required")
	}

	// All paths in a configuration must be absolute path.
	if c.ServerKeyPath != "" {
		path := c.ServerKeyPath
		if u, err := user.Current(); err == nil {
			path = strings.Replace(path, "~", u.HomeDir, 1)
		}

		path, err := filepath.Abs(path)
		if err != nil {
			return err
		}

		// generate host key if not exists.
		if _, err = os.Stat(path); err != nil {
			log.Infof("Create new host key '%s'", path)
			if err = c.generatePrivateKey(path); err != nil {
				return err
			}
		}
		c.ServerKeyPath = path

	} else {
		return fmt.Errorf("Server key file is required")
	}

	if c.PasswordFilePath != "" {
		path := c.PasswordFilePath
		if u, err := user.Current(); err == nil {
			path = strings.Replace(path, "~", u.HomeDir, 1)
		}

		path, err = filepath.Abs(path)
		if err != nil {
			return err
		}
		if _, err = os.Stat(path); err != nil {
			return fmt.Errorf("Password file '%s' is not found", c.PasswordFilePath)
		}
		c.PasswordFilePath = path
	}

	if c.AuthorizedKeysPath != "" {
		path := c.AuthorizedKeysPath
		if u, err := user.Current(); err == nil {
			path = strings.Replace(path, "~", u.HomeDir, 1)
		}

		path, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if _, err = os.Stat(path); err != nil {
			return fmt.Errorf("Authorized keys file '%s' is not found", c.AuthorizedKeysPath)
		}
		c.AuthorizedKeysPath = path

	} else {
		return fmt.Errorf("Authorized keys file is required")
	}

	// Default timeout
	if c.SwiftTimeout == 0 {
		c.SwiftTimeout = 180
	}

	return nil
}

// Generate ECDSA private key
func (c *Config) generatePrivateKey(path string) error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	encoded, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}

	data := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: encoded})

	return ioutil.WriteFile(path, data, 0600)
}

package main

import (
	"flag"
	"fmt"
	"reflect"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/urfave/cli"
)

func TestInitFromFile(t *testing.T) {
	filename := "misc/testing/test.toml"

	c := Config{}
	err := c.LoadFromFile(filename)
	if err != nil {
		t.Error(err)
	}

	targets := []string{
		"CreateContainerIfNotExists",
		"PasswordFilePath",
		"BindAddress",
		"AuthorizedKeysPath",
		"Container",
		"OsIdentityEndpoint",
		"OsUserID",
		"OsUsername",
		"OsPassword",
		"OsDomainID",
		"OsDomainName",
		"OsTenantID",
		"OsTenantName",
		"OsRegion",
	}

	if err := checkInitializedConfig(c, targets); err != nil {
		t.Error(err)
	}
}

func TestInitFromContext(t *testing.T) {
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	set.Bool("debug,d", false, "")
	set.String("container", "ojs-test-container", "")
	set.String("address", "127.0.0.1:20022", "")
	set.String("password-file", "misc/testing/dummypasswd", "")
	set.String("server-key", "server.key", "")
	set.String("authorized-keys", "misc/testing/authorized_keys", "")
	set.Parse([]string{
		"ojs-test-container",
	})
	ctx := cli.NewContext(cli.NewApp(), set, nil)

	c := Config{}
	err := c.LoadFromContext(ctx)
	if err != nil {
		t.Error(err)
	}

	targets := []string{
		"CreateContainerIfNotExists",
		"PasswordFilePath",
		"BindAddress",
		"AuthorizedKeysPath",
		"Container",
	}

	if err := checkInitializedConfig(c, targets); err != nil {
		t.Error(err)
	}
}

func checkInitializedConfig(c1 Config, targets []string) error {
	var c2 Config
	_, err := toml.DecodeFile("misc/testing/test.toml", &c2)
	if err != nil {
		return err
	}

	v1 := reflect.Indirect(reflect.ValueOf(c1))
	t1 := v1.Type()
	v2 := reflect.Indirect(reflect.ValueOf(c2))
	t2 := v2.Type()

	if t1.NumField() != t2.NumField() {
		return fmt.Errorf("Count of fields is not matched %d != %d", t1.NumField(), t2.NumField())
	}

	for _, field := range targets {
		var value1, value2 reflect.Value
		for i := 0; i < t1.NumField(); i++ {
			f := t1.Field(i)
			if f.Name == field {
				value1 = v1.Field(i)
			}
		}

		for i := 0; i < t2.NumField(); i++ {
			f := t2.Field(i)
			if f.Name == field {
				value2 = v2.Field(i)
			}
		}

		if value1.String() != value2.String() {
			return fmt.Errorf("Invalid value.. f=%s, %v != %v", field, value1, value2)
		}
	}
	return nil
}

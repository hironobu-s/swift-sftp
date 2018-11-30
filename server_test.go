package main

import (
	"encoding/pem"
	"io/ioutil"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	log = logrus.NewEntry(l)

	// config
	c := defaultConfigForTesting()

	// First, delete the container for testing
	s := NewSwift(c)
	if err := s.Init(); err != nil {
		panic(err)
	}
	s.DeleteContainer()
	s.CreateContainer()

	// run
	code := m.Run()

	// after testing
	s.DeleteContainer()

	os.Exit(code)
}

func defaultConfigForTesting() Config {
	// default value
	opts := ConfigInitOpts{
		Container:          "ojs-test-container",
		SourceAddress:      "127.0.0.1",
		Port:               10022,
		PasswordFilePath:   "",
		AuthorizedKeysPath: "~/.ssh/authorized_keys",
	}

	c := Config{}
	if err := c.Init(opts); err != nil {
		panic(err)
	}
	c.CreateContainerIfNotExists = true

	return c
}

var _swiftCache *Swift

func swiftForTesting() *Swift {
	if _swiftCache == nil {
		_swiftCache = NewSwift(defaultConfigForTesting())
		if err := _swiftCache.Init(); err != nil {
			panic(err)
		}
	}
	return _swiftCache
}

// -----------------------------------------------------

func TestInitServerHostkey(t *testing.T) {
	c := defaultConfigForTesting()
	sConf, _, err := initServer(c)
	if err != nil {
		t.Error(err)
		return
	}
	_ = sConf

	// make sure that server private key was generated
	data, err := ioutil.ReadFile(c.HostPrivateKeyPath)
	if err != nil {
		t.Error("Server private keyfile was not generated")
		return
	}
	pkey, _ := pem.Decode(data)
	if pkey.Type != "EC PRIVATE KEY" {
		t.Fatalf("Invalid private key")
		return
	}
}

func TestInitServerPasswdFile(t *testing.T) {
	sConf, _, err := initServer(defaultConfigForTesting())
	if err != nil {
		t.Error(err)
		return
	}

	// make sure that password authentication is disabled
	if sConf.PasswordCallback != nil {
		t.Fatal("Password authentication should be disabled")
	}
}

func TestInitServerPasswordAuth(t *testing.T) {
	c := defaultConfigForTesting()

	filename := "./passwd"
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		f.Close()
		os.Remove(filename)
	}()

	c.PasswordFilePath = filename

	// make sure that password authentication is dnabled after creating password file
	sConf, _, err := initServer(c)
	if sConf.PasswordCallback == nil {
		t.Fatal("Password authentication should be enabled")
	}
}

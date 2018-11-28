package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/k0kubun/pp"
	"github.com/pkg/sftp"
)

var (
	testfile = "testfile.txt"
	tmpfile  = "tmp_testfile.txt"
	s        *Swift
	fs       sftp.Handlers
)

func TestMain(m *testing.M) {
	c := Config{
		Container:                  "ojs-test-container",
		CreateContainerIfNotExists: true,
	}

	// First, delete the container
	s = NewSwift(c)
	s.Init()
	s.DeleteContainer()

	// Recreate swift for testing
	s = NewSwift(c)

	// fs
	fs = NewSwiftFS(s)

	// run
	code := m.Run()

	// after testing
	s.DeleteContainer()
	os.Remove(testfile)

	os.Exit(code)
}

func TestAuthFromEnv(t *testing.T) {
	if err := s.Init(); err != nil {
		fmt.Printf("%v", err)
		t.Fail()
	}
}

func TestList(t *testing.T) {
	ls, err := s.List()
	if err != nil {
		fmt.Printf("%v", err)
		t.Fail()
	}

	if len(ls) > 0 {
		pp.Printf("%v\n", ls)
		t.Errorf("%d objects exists on the object storage and then the test result might be incorrect.", len(ls))
	}
}

func TestPut(t *testing.T) {

	data := []byte("This is test data.")
	err := ioutil.WriteFile("testfile.txt", data, 0600)
	if err != nil {
		t.Errorf("%v", err)
		t.Fail()
	}

	err = s.Put(testfile, bytes.NewReader(data))
	if err != nil {
		t.Errorf("%v", err)
		t.Fail()
	}

	ls, err := s.List()
	if err != nil {
		t.Errorf("%v", err)
		t.Fail()
	}

	existTestfile := false
	for _, obj := range ls {
		if obj.Name == testfile {
			existTestfile = true
		} else if obj.Name == tmpfile {
			t.Errorf("Temporary file '%s' exists", tmpfile)
			t.Fail()
		}
	}

	if !existTestfile {
		t.Errorf("Does not exist testfile. '%s'", testfile)
	}
}
func TestGet(t *testing.T) {
	header, err := s.Get(testfile)
	if err != nil {
		t.Errorf("%v\n", err)
		t.Fail()
	} else if header == nil {
		t.Errorf("Couldn't get the header of the object")
		t.Fail()
	}
}

func TestDownload(t *testing.T) {
	obj, size, err := s.Download(testfile)
	if err != nil {
		t.Errorf("%v\n", err)
		t.Fail()
	}

	data1, _ := ioutil.ReadFile(testfile)
	data2, _ := ioutil.ReadAll(obj)

	if len(data1) != len(data2) {
		t.Errorf("Size missmatched between the test data and downloaded content. [%d != %d]\n", len(data1), len(data2))
	} else if int(size) != len(data2) {
		t.Errorf("Invalid size. [%d != %d]\n", size, len(data2))
		t.Fail()
	}
}

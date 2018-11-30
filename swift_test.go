package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

var (
	// testfile = "testfile.txt"
	// tmpfile  = "tmp_testfile.txt"
	tConfig Config
	tSwift  *Swift
)

func TestMain(m *testing.M) {
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	log = logrus.NewEntry(l)
	//enableDebugTransport()

	tConfig = Config{
		Container:                  "ojs-test-container",
		CreateContainerIfNotExists: true,
	}

	// First, delete the container for testing
	tSwift = NewSwift(tConfig)
	tSwift.Init()
	tSwift.DeleteContainer()
	tSwift.CreateContainer()

	// run
	code := m.Run()

	// after testing
	tSwift.DeleteContainer()

	os.Exit(code)
}

func TestAuthFromEnv(t *testing.T) {
	s := NewSwift(tConfig)
	if err := s.Init(); err != nil {
		fmt.Printf("%v", err)
		t.Fail()
	}
}

func TestPut(t *testing.T) {
	filename := "testdata.obj"

	data := []byte("This is test data.")
	err := ioutil.WriteFile(filename, data, 0600)
	if err != nil {
		t.Errorf("%v", err)
	}

	err = tSwift.Put(filename, bytes.NewReader(data))
	if err != nil {
		t.Errorf("%v", err)
	}

	ls, err := tSwift.List()
	if err != nil {
		t.Errorf("%v", err)
	}

	tmpfilename := "tmp-" + filename
	existTestfile := false
	for _, obj := range ls {
		if obj.Name == filename {
			existTestfile = true
		} else if obj.Name == tmpfilename {
			t.Errorf("Temporary file '%s' exists", tmpfilename)
		}
	}

	if !existTestfile {
		t.Errorf("Does not exist testfile. '%s'", filename)
	}
}
func TestGet(t *testing.T) {
	filename := "testdata.obj"

	header, err := tSwift.Get(filename)
	if err != nil {
		t.Errorf("%v\n", err)
		t.Fail()
	} else if header == nil {
		t.Errorf("Couldn't get the header of the object")
		t.Fail()
	}
}

func TestDownload(t *testing.T) {
	filename := "testdata.obj"
	// remove test file
	defer func() {
		os.Remove(filename)
	}()

	obj, size, err := tSwift.Download(filename)
	if err != nil {
		t.Errorf("%v\n", err)
		t.Fail()
	}

	data1, _ := ioutil.ReadFile(filename)
	data2, _ := ioutil.ReadAll(obj)

	if len(data1) != len(data2) {
		t.Errorf("Size missmatched between the test data and downloaded content. [%d != %d]\n", len(data1), len(data2))
	} else if int(size) != len(data2) {
		t.Errorf("Invalid size. [%d != %d]\n", size, len(data2))
		t.Fail()
	}
}

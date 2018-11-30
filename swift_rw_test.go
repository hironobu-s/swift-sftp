package main

import (
	"bytes"
	"crypto/sha256"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func prepareTestObject() (tmpfile string, data []byte, err error) {
	tmpfile = "download-test.obj"

	data = []byte(strings.Repeat("data", 1000))
	if err = ioutil.WriteFile(tmpfile, data, 0600); err != nil {
		return "", nil, err
	}

	if err = tSwift.Put(tmpfile, bytes.NewBuffer(data)); err != nil {
		return "", nil, err
	}

	return tmpfile, data, nil
}

func TestReaderDownload(t *testing.T) {
	filename, data, err := prepareTestObject()
	defer func() {
		os.Remove(filename)
	}()

	if err != nil {
		t.Fatal(err)
		return
	}

	f := &SwiftFile{
		objectname: filename,
		size:       0,
		modtime:    time.Now(),
	}
	r := swiftReader{swift: tSwift, sf: f}

	err = r.download(f.Name())
	if err != nil {
		t.Error(err)
		return
	}

	downloaded, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Error(err)
		return
	}

	hash1 := sha256.Sum256(data)
	hash2 := sha256.Sum256(downloaded)
	if !reflect.DeepEqual(hash1, hash2) {
		t.Errorf("Hash values don't matche %s, %s", hash1, hash2)
	}
}

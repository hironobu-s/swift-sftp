package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"
)

func generateTestFile(filename string, size int64) (data []byte, err error) {
	data = make([]byte, size)
	w := bytes.NewBuffer(data)
	_, err = io.CopyN(w, rand.Reader, size)

	if err != nil {
		return nil, err
	}

	if err = ioutil.WriteFile(filename, data, 0600); err != nil {
		return nil, err
	}

	return data, nil
}

func generateTestObject(filename string, size int64) (data []byte, err error) {
	data, err = generateTestFile(filename, size)
	if err != nil {
		return nil, err
	}

	if err = tSwift.Put(filename, bytes.NewBuffer(data)); err != nil {
		return nil, err
	}

	return data, nil
}

func TestReaderDownload(t *testing.T) {
	filename := "reader-test.dat"
	data, err := generateTestObject(filename, 1024*1024)
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

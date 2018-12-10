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
	s := swiftForTesting()

	data, err = generateTestFile(filename, size)
	if err != nil {
		return nil, err
	}

	if err = s.Put(filename, bytes.NewBuffer(data)); err != nil {
		return nil, err
	}

	return data, nil
}

func TestReaderDownload(t *testing.T) {
	s := swiftForTesting()

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
	r := swiftReader{swift: s, sf: f}

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

func TestWriterUpload(t *testing.T) {
	s := swiftForTesting()

	filename := "writer-test.dat"
	data, err := generateTestFile(filename, 1024*1024)
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
	w := swiftWriter{swift: s, sf: f}

	r := bytes.NewBuffer(data)
	var offset int64 = 0
	buf := make([]byte, 128)
	for true {
		n, err := r.Read(buf)
		if err != nil {
			break
		}
		w.WriteAt(buf, offset)
		offset += int64(n)
	}

	// object wil be uploaded if close() is called
	w.Close()

	// download uploaded object and comparet it with local file
	u, _, err := s.Download(filename)
	if err != nil {
		t.Error(err)
	}

	uploaded, _ := ioutil.ReadAll(u)
	if bytes.Compare(uploaded, data) != 0 {
		t.Errorf("Both contents does't matche")
	}

	if _, err := os.Stat(w.tmpfile.Name()); err == nil {
		t.Errorf("Temporary file is sill exist")
	}
}

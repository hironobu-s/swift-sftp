package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"sort"

	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/pkg/sftp"
)

func TestFileread(t *testing.T) {
	filename := "fileread-test.dat"
	defer func() {
		os.Remove(filename)
	}()

	data, err := generateTestObject(filename, 1024*1024)
	if err != nil {
		t.Error(err)
	}

	fs := NewSwiftFS(tSwift)
	req := sftp.NewRequest("Read", "/"+filename)
	r, err := fs.Fileread(req)
	if err != nil {
		t.Error(err)
	}
	_ = data
	_ = r
}

func TestFilewrite(t *testing.T) {
	filename := "filewrite-test.dat"
	defer func() {
		os.Remove(filename)
	}()

	data, err := generateTestFile(filename, 1024*1024)

	fs := NewSwiftFS(tSwift)
	req := sftp.NewRequest("Write", "/"+filename)
	sw, err := fs.Filewrite(req)
	if err != nil {
		t.Error(err)
	}

	reader := bytes.NewReader(data)
	bs := 128
	offset := 0
	for offset < reader.Len() {
		buf := make([]byte, bs)
		reader.ReadAt(buf, int64(offset))

		n, err := sw.WriteAt(buf, int64(offset))
		if n != bs {
			t.Errorf("byte size missmatch %d, %d", bs, n)
			break
		}
		if err != io.EOF && err != nil {
			t.Error(err)
			break
		}
		offset += n
	}

	// Data written by following loop will upload when calls Close()
	c, _ := sw.(io.Closer)
	c.Close()

	r, _, err := tSwift.Download(filename)
	if err != nil {
		t.Error(err)
	}

	content, _ := ioutil.ReadAll(r)
	if !reflect.DeepEqual(content, data) {
		t.Errorf("Wrong downloaded data")
	}
}

func TestFilecmd(t *testing.T) {
	filename := "filecmd-rename-test.dat"
	targetName := "filecmd-rename-target-test.dat"
	defer func() {
		os.Remove(filename)
	}()

	data, err := generateTestObject(filename, 128)

	req := sftp.NewRequest("Rename", "/"+filename)
	req.Target = "/" + targetName

	fs := NewSwiftFS(tSwift)
	if err = fs.Filecmd(req); err != nil {
		t.Error(err)
	}

	if _, err = tSwift.Get(filename); err == nil {
		t.Error("Original file that should be deleted exists")
	}
	_, ok := err.(gophercloud.ErrDefault404)
	if !ok {
		t.Error("Original file that should be deleted exists")
	}

	r, _, err := tSwift.Download(targetName)
	content, _ := ioutil.ReadAll(r)
	if !reflect.DeepEqual(content, data) {
		t.Errorf("Wrong downloaded data")
	}

	if err = tSwift.Delete(targetName); err != nil {
		t.Error(err)
	}

	if _, err = tSwift.Get(targetName); err == nil {
		t.Error("File that should be deleted exists")
	}
	_, ok = err.(gophercloud.ErrDefault404)
	if !ok {
		t.Error("File that should be deleted exists")
	}
}

func TestFilelist(t *testing.T) {
	files := []string{
		"filelist-foo-test.dat",
		"filelist-bar-test.dat",
		"filelist-baz-test.dat",
	}
	sort.Strings(files)

	defer func() {
		for _, name := range files {
			os.Remove(name)
			tSwift.Delete(name)
		}
	}()

	for i := 0; i < len(files); i++ {
		_, err := generateTestObject(files[i], 128)
		if err != nil {
			t.Error(err)
		}
	}

	// list
	fs := NewSwiftFS(tSwift)

	req := sftp.NewRequest("List", "/")
	l, err := fs.Filelist(req)
	if err != nil {
		t.Error(err)
	}

	list := make([]os.FileInfo, len(files))
	n, err := l.ListAt(list, 0)
	if err != nil && err != io.EOF {
		t.Error(err)
	} else if n != len(files) {
		t.Error("Count of list is different from count of test data")
		return
	}

	sort.Slice(list, func(i int, j int) bool {
		return strings.Compare(list[i].Name(), list[j].Name()) == -1
	})

	for i := 0; i < len(files); i++ {
		if list[i].Name() != files[i] {
			t.Errorf("Filename missmatch [i=%d, %s != %s]", i, list[i].Name(), files[i])
		}
	}

	// stat
	for _, name := range files {
		req := sftp.NewRequest("Stat", "/"+name)
		s, err := fs.Filelist(req)
		if err != nil {
			t.Error(err)
		}

		list := make([]os.FileInfo, 1)
		_, err = s.ListAt(list, 0)
		if err != nil {
			t.Error(err)
			break
		}
		if list[0].Name() != name {
			t.Errorf("Filename missmatch [%s != %s]", list[0].Name(), name)
		}
	}
}

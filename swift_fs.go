package main

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"github.com/sirupsen/logrus"
)

const (
	Delimiter = "/" // Delimiter is used to split object names.
)

// SwiftFS implements sftp.Handlers interface.
type SwiftFS struct {
	log *logrus.Entry

	lock    sync.Mutex
	mockErr error

	swift        *Swift
	waitReadings []*SwiftFile
	waitWritings []*SwiftFile
}

func NewSwiftFS(s *Swift) *SwiftFS {
	fs := &SwiftFS{
		log:   log,
		swift: s,
	}

	return fs
}

func (fs *SwiftFS) SetLogger(clog *logrus.Entry) {
	fs.log = clog
}

func (fs *SwiftFS) debug(r *sftp.Request) {
	if r.Target != "" {
		fs.log.Debugf("%s %s (target=%s)", r.Method, r.Filepath, r.Target)
	} else {
		fs.log.Debugf("%s %s", r.Method, r.Filepath)
	}
}

func (fs *SwiftFS) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	fs.debug(r)

	fs.lock.Lock()
	defer fs.lock.Unlock()

	f, err := fs.lookup(r.Filepath)
	if err != nil {
		return nil, err

	} else if f == nil {
		return nil, fmt.Errorf("File not found. [%s]", r.Filepath)
	}

	to := time.Duration(fs.swift.config.SwiftTimeout) * time.Second
	return &swiftReader{
		log:     fs.log,
		swift:   fs.swift,
		sf:      f,
		timeout: to,
	}, nil
}

func (fs *SwiftFS) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	fs.debug(r)

	fs.lock.Lock()
	defer fs.lock.Unlock()

	f := &SwiftFile{
		objectname: r.Filepath[1:], // strip slash
		size:       0,
		modtime:    time.Now(),
		symlink:    "",
		isdir:      false,
	}

	to := time.Duration(fs.swift.config.SwiftTimeout) * time.Second
	return &swiftWriter{
		log:     fs.log,
		swift:   fs.swift,
		sf:      f,
		timeout: to,
	}, nil
}

func (fs *SwiftFS) Filecmd(r *sftp.Request) error {
	fs.debug(r)

	if fs.mockErr != nil {
		return fs.mockErr
	}
	fs.lock.Lock()
	defer fs.lock.Unlock()

	f, err := fs.lookup(r.Filepath)
	if err != nil {
		return err
	}

	switch r.Method {
	case "Rename":

		tf := SwiftFile{
			objectname: r.Target,
		}
		target := &SwiftFile{
			objectname: tf.Name(),
			size:       0,
			modtime:    time.Now(),
			symlink:    "",
			isdir:      false,
		}
		fs.log.Infof("Rename '%s' to '%s'", f.Name(), target.Name())

		return fs.swift.Rename(f.Name(), target.Name())

	case "Remove":
		err = fs.swift.Delete(f.Name())
		if err != nil {
			return err
		}
		fs.log.Infof("Remove '%s'", f.Name())

	default:
		fs.log.Debugf("Unsupported operation [method=%s, file=%s]", r.Method, r.Target)
	}
	return nil
}

func (fs *SwiftFS) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	fs.debug(r)

	if fs.mockErr != nil {
		return nil, fs.mockErr
	}
	fs.lock.Lock()
	defer fs.lock.Unlock()

	switch r.Method {
	case "List":
		files, err := fs.walk(r.Filepath)
		if err != nil {
			return nil, err
		}

		list := make([]os.FileInfo, 0, len(files))
		for _, f := range files {
			list = append(list, f)
		}
		return listerat(list), nil

	case "Stat":
		f, err := fs.lookup(r.Filepath)
		if err != nil {
			return nil, err
		}
		if f != nil {
			return listerat([]os.FileInfo{f}), nil
		} else {
			return listerat([]os.FileInfo{}), nil
		}

	default:
		fs.log.Debugf("Unsupported operation [method=%s, file=%s]", r.Method, r.Target)
	}
	return nil, nil
}

func (fs *SwiftFS) filepath2object(path string) string {
	return path[1:]
}
func (fs *SwiftFS) object2filepath(name string) string {
	return Delimiter + name
}

// Return SwiftFile objects in the specific directory
func (fs *SwiftFS) walk(dirname string) ([]*SwiftFile, error) {
	files, err := fs.allFiles()
	if err != nil {
		return nil, err
	}

	list := make([]*SwiftFile, 0, len(files))
	for _, f := range files {
		if f.Abs() != dirname && f.Dir() == dirname {
			list = append(list, f)
		}
	}
	return list, nil
}

// Return SwiftFile object with the path
func (fs *SwiftFS) lookup(path string) (*SwiftFile, error) {
	// root path is not on the object storage and return it manually.
	if path == "/" {
		f := &SwiftFile{
			objectname: "",
			modtime:    time.Now(),
			isdir:      true,
		}
		return f, nil
	}

	name := fs.filepath2object(path)
	header, err := fs.swift.Get(name)
	if err != nil {
		return nil, err
	}

	f := &SwiftFile{
		objectname: name,
		size:       header.ContentLength,
		modtime:    header.LastModified,
		symlink:    "",
		isdir:      false,
	}
	return f, nil
}

// To synchronize objects on object storage and fs.files
func (fs *SwiftFS) allFiles() ([]*SwiftFile, error) {
	fs.log.Debugf("Updating file list...")

	// Get object list from object storage
	objs, err := fs.swift.List()
	if err != nil {
		return nil, err
	}

	files := make([]*SwiftFile, len(objs))
	for i, obj := range objs {
		files[i] = &SwiftFile{
			objectname: obj.Name,
			size:       obj.Bytes,
			modtime:    obj.LastModified,
			isdir:      false,
		}
	}
	return files, nil
}

// Modeled after strings.Reader's ReadAt() implementation
type listerat []os.FileInfo

func (f listerat) ListAt(ls []os.FileInfo, offset int64) (int, error) {
	var n int
	if offset >= int64(len(f)) {
		return 0, io.EOF
	}
	n = copy(ls, f[offset:])
	if n < len(ls) {
		return n, io.EOF
	}
	return n, nil
}

package main

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
)

// swiftFS implements sftp.Handlers interface.
type swiftFS struct {
	lock    sync.Mutex
	mockErr error
	swift   *Swift
}

func NewSwiftFS(s *Swift) sftp.Handlers {
	h := &swiftFS{
		swift: s,
	}
	return sftp.Handlers{h, h, h, h}
}

func (fs *swiftFS) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	log.Debug("Calling Fileread() in swiftFS")

	content, err := fs.swift.Download(fs.convertPathToObjectName(r.Filepath))
	if err != nil {
		return nil, err
	}

	return &UnbufferedReader{R: content}, err
}

func (fs *swiftFS) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	log.Debug("Calling Filewrite() in swiftFS")

	return nil, nil
}

func (fs *swiftFS) Filecmd(r *sftp.Request) error {
	log.Debug("Calling Filecmd() in swiftFS")
	return nil
}

func (fs *swiftFS) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	log.Debug("Calling Filelist() in swiftFS")

	if fs.mockErr != nil {
		return nil, fs.mockErr
	}
	fs.lock.Lock()
	defer fs.lock.Unlock()

	switch r.Method {
	case "List":
		return listerat(fs.findAll()), nil

	case "Stat":
		f := fs.findOne(r.Filepath)

		if f != nil {
			return listerat([]os.FileInfo{f}), nil
		} else {
			return listerat([]os.FileInfo{}), nil
		}

	case "Readlink":
		return nil, nil
	}
	return nil, nil
}

func (fs *swiftFS) convertPathToObjectName(name string) (objname string) {
	return filepath.Base(name)
}

func (fs *swiftFS) findOne(name string) os.FileInfo {
	log.Debugf("find one [%s]", name)

	var f *swiftFile
	if name == "/" {
		f = &swiftFile{
			name:    "/",
			modtime: time.Now(),
			isdir:   true,
		}

	} else {
		header, err := fs.swift.Get(fs.convertPathToObjectName(name))
		if err != nil {
			// Not found
			return nil
		}

		f = &swiftFile{
			name:    name,
			size:    header.ContentLength,
			modtime: header.LastModified,
			isdir:   false,
		}
	}
	return f
}

func (fs *swiftFS) findAll() (list []os.FileInfo) {
	log.Debugf("find all")

	objs, err := fs.swift.List()
	if err != nil {
		return list
	}

	list = make([]os.FileInfo, len(objs))
	for i, obj := range objs {
		f := &swiftFile{}
		f.name = obj.Name
		f.size = obj.Bytes
		f.modtime = obj.LastModified
		f.isdir = false
		f.content = nil
		list[i] = f
	}
	return list
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

// swiftFile implements os.FileInfo, os.Reader, os.Writer interfaces.
// There interfaces are necessary for sftp.Handlers.
type swiftFile struct {
	name        string
	size        int64
	modtime     time.Time
	symlink     string
	isdir       bool
	content     []byte
	contentLock sync.RWMutex
}

func (f *swiftFile) Name() string {
	return f.name
}

func (f *swiftFile) Size() int64 {
	return f.size
}
func (f *swiftFile) Mode() os.FileMode {
	return os.FileMode(0666)
}
func (f *swiftFile) ModTime() time.Time { return f.modtime }
func (f *swiftFile) IsDir() bool        { return f.isdir }
func (f *swiftFile) Sys() interface{} {
	return nil
}

// Read/Write
func (f *swiftFile) ReaderAt() (io.ReaderAt, error) {
	return nil, os.ErrInvalid
}

func (f *swiftFile) WriterAt() (io.WriterAt, error) {
	return nil, os.ErrInvalid
}
func (f *swiftFile) WriteAt(p []byte, off int64) (int, error) {
	return 0, nil
}

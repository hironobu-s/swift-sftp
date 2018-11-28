package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
)

const (
	Separator = "/" // Separator of the object on the object storage
)

// SwiftFS implements sftp.Handlers interface.
type SwiftFS struct {
	lock    sync.Mutex
	mockErr error

	swift        *Swift
	waitReadings []*SwiftFile
	waitWritings []*SwiftFile
}

func NewSwiftFS(s *Swift) *SwiftFS {
	fs := &SwiftFS{
		swift: s,
	}

	// To initialize waiting slices
	fs.SyncWaitingFiles()

	return fs
}

func (fs *SwiftFS) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	log.Debugf("file read, method=%s filepath=%s", r.Method, r.Filepath)

	fs.lock.Lock()
	defer fs.lock.Unlock()

	if err := fs.SyncWaitingFiles(); err != nil {
		return nil, err
	}

	f, err := fs.lookup(r.Filepath)
	if err != nil {
		return nil, err

	} else if f == nil {
		return nil, fmt.Errorf("File not found. [%s]", r.Filepath)
	}

	// // append f to waiting list
	// fs.waitReadings = append(fs.waitReadings, f)

	return &swiftReadWriter{
		swift: fs.swift,
		sf:    f,
	}, nil
}

func (fs *SwiftFS) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	log.Debugf("file write, method=%s filepath=%s", r.Method, r.Filepath)

	fs.lock.Lock()
	defer fs.lock.Unlock()

	if err := fs.SyncWaitingFiles(); err != nil {
		return nil, err
	}

	f := &SwiftFile{
		objectname: r.Filepath[1:], // strip slash
		size:       0,
		modtime:    time.Now(),
		symlink:    "",
		isdir:      false,
	}

	return &swiftReadWriter{
		swift: fs.swift,
		sf:    f,
	}, nil
}

func (fs *SwiftFS) Filecmd(r *sftp.Request) error {
	log.Debug("Calling Filecmd() in SwiftFS")
	return nil
}

func (fs *SwiftFS) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	log.Debugf("file list method=%s filepath=%s", r.Method, r.Filepath)

	if fs.mockErr != nil {
		return nil, fs.mockErr
	}
	fs.lock.Lock()
	defer fs.lock.Unlock()

	if err := fs.SyncWaitingFiles(); err != nil {
		return nil, err
	}

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

	case "Readlink":
		return nil, nil
	}
	return nil, nil
}

func (fs *SwiftFS) filepath2object(path string) string {
	return path[1:]
}
func (fs *SwiftFS) object2filepath(name string) string {
	return Separator + name
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
	log.Debugf("Updating file list...")

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

// SwiftFile implements os.FileInfo interfaces.
// There interfaces are necessary for sftp.Handlers.
type SwiftFile struct {
	objectname string
	size       int64
	modtime    time.Time
	symlink    string
	isdir      bool

	tmpFile *os.File
}

func (f *SwiftFile) Abs() string {
	return f.Dir() + f.objectname
}

func (f *SwiftFile) Dir() string {
	if strings.HasSuffix(f.objectname, Separator) {
		// f.name is directory name
		return f.objectname

	} else if !strings.Contains(f.objectname, Separator) {
		// f.objectname is the file on root file path
		return Separator

	} else {
		pos := strings.LastIndex(f.objectname, Separator)
		return f.objectname[:pos+1]
	}
}

func (f *SwiftFile) TempFileName() string {
	t := time.Now().Format(time.RFC3339Nano)
	h := sha256.Sum256([]byte(t))
	return filepath.Join(os.TempDir(), "ojs-"+hex.EncodeToString(h[:]))
}

// io.Fileinfo interface
func (f *SwiftFile) Name() string {
	pos := strings.LastIndex(f.objectname, Separator)
	return f.objectname[pos+1:]
}

func (f *SwiftFile) Size() int64 {
	return f.size
}

func (f *SwiftFile) Mode() os.FileMode {
	return os.FileMode(0666)
}

func (f *SwiftFile) ModTime() time.Time {
	return f.modtime
}

func (f *SwiftFile) IsDir() bool {
	return f.isdir
}

func (f *SwiftFile) Sys() interface{} {
	// return dummy stat data
	type Timespec struct {
		Sec  int64
		Nsec int64
	}

	type stat struct {
		Dev       uint64
		Ino       uint64
		Nlink     uint64
		Mode      uint32
		Uid       uint32
		Gid       uint32
		X__pad0   int32
		Rdev      uint64
		Size      int64
		Blksize   int64
		Blocks    int64
		Atim      Timespec
		Mtim      Timespec
		Ctim      Timespec
		X__unused [3]int64
	}
	return stat{}
}

// swiftReadWriter implements both interfaces, io.ReadAt and io.WriteAt.
type swiftReadWriter struct {
	swift   *Swift
	sf      *SwiftFile
	tmpfile *os.File
}

func (rw *swiftReadWriter) ReadAt(p []byte, off int64) (n int, err error) {
	if rw.tmpfile == nil {
		log.Debugf("Download content from object storage. [name=%s]", rw.sf.Name())

		body, size, err := rw.swift.Download(rw.sf.Name())
		if err != nil {
			return 0, err
		}
		defer body.Close()
		log.Debugf("Completed downloading. [size=%d]", size)

		fname := rw.sf.TempFileName()
		log.Debugf("Create tmpfile to read. [%s]", fname)

		w, err := os.OpenFile(fname, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
		if err != nil {
			log.Warnf("%v", err.Error())
			return 0, err
		}
		defer w.Close()

		_, err = io.Copy(w, body)
		if err != nil {
			log.Warnf("%v", err.Error())
			return 0, err
		}

		// Reopen tmporary file for reading
		// Do not need to call tmpfile.Close(). It'll be called in swiftReadWriter.Close()
		rw.tmpfile, err = os.OpenFile(fname, os.O_RDONLY, 0000)
		if err != nil {
			log.Warnf("%v", err.Error())
			return 0, err
		}
	}

	log.Debugf("Read from tmpfile, offset=%d len=%d", off, len(p))

	return rw.tmpfile.ReadAt(p, off)
}

func (rw *swiftReadWriter) WriteAt(p []byte, off int64) (n int, err error) {
	if rw.tmpfile == nil {
		fname := rw.sf.TempFileName()
		log.Debugf("Create tmpfile to write. [%s]", fname)

		// Do not need to call tmpfile.Close(). It'll be called in swiftReadWriter.Close()
		rw.tmpfile, err = os.OpenFile(fname, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
		if err != nil {
			log.Warnf("%v", err.Error())
			return 0, err
		}
	}

	log.Debugf("Write to tmpfile, offset=%d len=%d", off, len(p))

	// write buffer to the temporary file
	_, err = rw.tmpfile.WriteAt(p, off)
	if err != nil {
		log.Warnf("%v", err.Error())
		return 0, err
	}

	return len(p), nil
}

func (rw *swiftReadWriter) Close() error {
	log.Debugf("Close and delete tmpfile")

	rw.tmpfile.Close()
	os.Remove(rw.tmpfile.Name())

	return nil
}

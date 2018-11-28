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

	// append f to waiting list
	fs.waitReadings = append(fs.waitReadings, f)

	return f.Reader()
}

func (fs *SwiftFS) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	log.Debugf("file write, method=%s filepath=%s", r.Method, r.Filepath)

	fs.lock.Lock()
	defer fs.lock.Unlock()

	if err := fs.SyncWaitingFiles(); err != nil {
		return nil, err
	}

	f := &SwiftFile{
		swift:      fs.swift,
		objectname: r.Filepath[1:], // strip slash
		size:       0,
		modtime:    time.Now(),
		symlink:    "",
		isdir:      false,
	}

	// append f to waiting list
	fs.waitWritings = append(fs.waitWritings, f)

	return f.Writer()
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

func (fs *SwiftFS) SyncWaitingFiles() error {
	for _, f := range fs.waitReadings {
		log.Debugf("Delete temporary file. [%s]", f.tmpFile.Name())
		f.tmpFile.Close()
		os.Remove(f.tmpFile.Name())
		f.tmpFile = nil
	}

	for _, f := range fs.waitWritings {
		log.Debugf("Upload content to object storage. [name=%s, size=%d]", f.Name(), f.Size())

		fn, err := os.OpenFile(f.tmpFile.Name(), os.O_RDONLY, 0600)
		if err != nil {
			return err
		}

		err = fs.swift.Put(f.objectname, fn)
		if err != nil {
			fn.Close()
			log.Warnf("Upload error. %v", err)
			return err
		}
		fn.Close()

		os.Remove(f.tmpFile.Name())
		f.tmpFile = nil
		log.Debugf("Completed uploading and deleted a temprary file")
	}

	fs.waitReadings = make([]*SwiftFile, 0, 10)
	fs.waitWritings = make([]*SwiftFile, 0, 10)
	return nil
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
			swift: fs.swift,

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
		swift:      fs.swift,
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
			swift:      fs.swift,
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

// SwiftFile implements os.FileInfo, os.Reader, os.Writer interfaces.
// There interfaces are necessary for sftp.Handlers.
type SwiftFile struct {
	swift *Swift

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

func (f *SwiftFile) Reader() (io.ReaderAt, error) {
	log.Debugf("Download content from object storage. [name=%s]", f.Name())

	body, size, err := f.swift.Download(f.Name())
	if err != nil {
		return nil, err
	}
	defer body.Close()

	log.Debugf("Completed downloading. [size=%d]", size)

	var tmpfile *os.File
	tmpfilename := f.TempFileName()
	log.Debugf("Temporary file for reading is '%s'.", tmpfilename)

	wf, err := os.OpenFile(tmpfilename, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600)
	if err != nil {
		log.Warnf("%v", err.Error())
		return nil, err
	}

	_, err = io.Copy(wf, body)
	if err != nil {
		// close
		wf.Close()

		// delete tmp file
		os.Remove(tmpfilename)

		log.Warnf("%v", err.Error())
		return nil, err
	}
	wf.Close()

	// reopen tmporary file for reading
	// Do not need to tmpfile.Close(). It'll be called in SwiftFS.sync()
	tmpfile, err = os.OpenFile(tmpfilename, os.O_RDONLY, 0000)
	if err != nil {
		log.Warnf("%v", err.Error())
		return nil, err
	}

	f.tmpFile = tmpfile

	return f.tmpFile, nil
}

func (f *SwiftFile) Writer() (io.WriterAt, error) {
	tmpfilename := f.TempFileName()
	log.Debugf("Temporary file for writing is '%s'.", tmpfilename)

	// Do not need to call wf.Close(). It'll be called in SwiftFS.sync()
	wf, err := os.OpenFile(tmpfilename, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600)
	if err != nil {
		log.Warnf("%v", err.Error())
		return nil, err
	}

	f.tmpFile = wf

	return f.tmpFile, nil
}

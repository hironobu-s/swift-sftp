package main

import (
	"os"
	"strings"
	"time"
)

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
	if strings.HasSuffix(f.objectname, Delimiter) {
		// f.name is directory name
		return f.objectname

	} else if !strings.Contains(f.objectname, Delimiter) {
		// f.objectname is the file on root file path
		return Delimiter

	} else {
		pos := strings.LastIndex(f.objectname, Delimiter)
		return f.objectname[:pos+1]
	}
}

// io.Fileinfo interface
func (f *SwiftFile) Name() string {
	pos := strings.LastIndex(f.objectname, Delimiter)
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

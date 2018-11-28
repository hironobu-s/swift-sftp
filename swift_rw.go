package main

import (
	"io"
	"os"

	log "github.com/sirupsen/logrus"
)

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

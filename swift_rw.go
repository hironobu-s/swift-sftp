package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// swiftReader implements io.ReadAt interface
type swiftReader struct {
	// Required to set in initialized
	log     *logrus.Entry
	swift   *Swift
	sf      *SwiftFile
	timeout time.Duration

	// Not required
	m            sync.Mutex
	tmpfile      *os.File
	downloadErr  error
	downloadSize int64
	readSize     int64

	afterClosed func(r *swiftReader)
}

func (r *swiftReader) Begin() (err error) {
	r.log.Debugf("Send '%s' (size=%d) to client", r.sf.Name(), r.sf.Size())

	// Download size
	headers, err := r.swift.Get(r.sf.Name())
	if err != nil {
		return err
	}
	r.downloadSize = headers.ContentLength
	if r.downloadSize == 0 {
		return fmt.Errorf("Couldn't detect download size (Missing Content-length header).")
	}

	// Create tmpfile
	fname, err := createTmpFile()
	if err != nil {
		return err
	}

	// Open tmpfile
	r.tmpfile, err = os.OpenFile(fname, os.O_RDONLY, 0000)
	if err != nil {
		r.log.Warnf("Couldn't open tmpfile. [%v]", err.Error())
		return err
	}

	// start download
	go func() {
		if err := r.download(fname); err != nil {
			r.downloadErr = err
		}
	}()
	return nil
}

func (r *swiftReader) download(tmpFileName string) (err error) {
	r.log.Debugf("Create tmpfile. [%s]", tmpFileName)
	fw, err := os.OpenFile(tmpFileName, os.O_WRONLY|os.O_TRUNC, 0000)
	if err != nil {
		r.log.Warnf("%v", err.Error())
		return err
	}
	defer fw.Close()

	body, size, err := r.swift.Download(r.sf.Name())
	if err != nil {
		return err
	}
	defer body.Close()

	r.log.Debugf("Download '%s' (size=%d) from Object Storage", r.sf.Name(), size)
	_, err = io.Copy(fw, body)
	if err != nil {
		r.log.Warnf("Error occured during copying [%v]", err.Error())
		return err
	}
	r.log.Debugf("Download completed")

	return nil
}

func (r *swiftReader) ReadAt(p []byte, off int64) (n int, err error) {
	start := time.Now()
	for {
		n, err = r.tmpfile.ReadAt(p, off)
		if n != 0 {
			r.readSize += int64(n)
			return n, err

		} else if r.readSize == r.downloadSize {
			r.log.Debugf("Send EOF to client. [%s]", r.sf.Name())
			return n, io.EOF
		}

		time.Sleep(100 * time.Millisecond)
		r.log.Debugf("Wait for downloading. [%s] ", r.sf.Name())

		if time.Now().Sub(start) > r.timeout {
			r.log.Warnf("Download timeout. [%s]", r.sf.Name())
			break
		}
	}

	r.downloadErr = errors.New("Timeout for downloading")
	return -1, r.downloadErr
}

func (r *swiftReader) Close() error {
	if r.afterClosed != nil {
		defer r.afterClosed(r)
	}

	// remove temporary file
	if r.tmpfile != nil {
		os.Remove(r.tmpfile.Name())
	}

	return nil
}

// swiftWriter implements io.WriteAt interface
type swiftWriter struct {
	// Required to set in initialized
	log     *logrus.Entry
	swift   *Swift
	sf      *SwiftFile
	timeout time.Duration

	// Not required
	tmpfile        *os.File
	uploadComplete bool
	uploadErr      error

	afterClosed func(w *swiftWriter)
}

func (w *swiftWriter) Begin() (err error) {
	w.log.Debugf("Receive '%s' from client", w.sf.Name())

	// Create tmpfile
	fname, err := createTmpFile()
	if err != nil {
		return err
	}

	// Open tmpfile
	w.tmpfile, err = os.OpenFile(fname, os.O_WRONLY, 0000)
	if err != nil {
		w.log.Warnf("Couldn't open tmpfile. [%v]", err.Error())
		return err
	}
	return nil
}

func (w *swiftWriter) upload() (err error) {
	fname := w.tmpfile.Name()
	w.log.Debugf("Upload: create tmpfile. [%s]", fname)
	fr, err := os.OpenFile(fname, os.O_RDONLY, 000)
	if err != nil {
		w.log.Warnf("%v", err.Error())
		return err
	}
	defer fr.Close()

	return w.swift.Put(w.sf.Name(), fr)
}

func (w *swiftWriter) WriteAt(p []byte, off int64) (n int, err error) {
	n, err = w.tmpfile.WriteAt(p, off)
	if err != nil {
		w.log.Debugf("%v", err)
	}
	return n, err
}

func (w *swiftWriter) Close() error {
	if w.afterClosed != nil {
		defer w.afterClosed(w)
	}

	// start uploading
	if w.tmpfile != nil {
		s, err := w.tmpfile.Stat()
		if err != nil {
			return err
		}

		w.log.Debugf("Upload '%s' (size=%d) to Object Storage", w.sf.Name(), s.Size())

		//go func() {
		defer func() {
			w.uploadComplete = true
		}()

		if err := w.upload(); err != nil {
			w.uploadErr = err
			w.log.Debugf("Upload: complete with error. [%v]", err)
		}

		// remove temporary file
		os.Remove(w.tmpfile.Name())

		if w.uploadErr != nil {
			return w.uploadErr
		}
		w.log.Debugf("'%s' was uploaded successfully", w.sf.Name())

		//}()
	}

	return nil
}

func createTmpFile() (string, error) {
	t := time.Now().Format(time.RFC3339Nano)
	h := sha256.Sum256([]byte(t))
	fname := filepath.Join(os.TempDir(), "ojs-"+hex.EncodeToString(h[:]))

	f, err := os.OpenFile(fname, os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return "", err
	}
	f.Close()

	return fname, nil
}

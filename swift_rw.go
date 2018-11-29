package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
)

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

type progressFunc func(tmpfile string) error

// swiftReadWriter implements both interfaces, io.ReadAt and io.WriteAt.
type swiftReader struct {
	swift *Swift
	sf    *SwiftFile

	tmpfile          *os.File
	downloadComplete bool
	downloadErr      error
}

func (r *swiftReader) download(tmpFileName string) (err error) {
	log.Debugf("Download: create tmpfile. [%s]", tmpFileName)

	fw, err := os.OpenFile(tmpFileName, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
	if err != nil {
		log.Warnf("%v", err.Error())
		return err
	}
	defer fw.Close()

	log.Debugf("Download: get '%s' from object storage.", r.sf.Name())
	body, size, err := r.swift.Download(r.sf.Name())
	if err != nil {
		return err
	}
	defer body.Close()

	log.Debugf("Download: start [size=%d]", size)
	_, err = io.Copy(fw, body)
	if err != nil {
		log.Warnf("Download: error occured during copying [%v]", err.Error())
		return err
	}
	log.Debugf("Download: complete")

	return nil
}

func (r *swiftReader) ReadAt(p []byte, off int64) (n int, err error) {
	if r.tmpfile == nil {
		// Create tmpfile
		fname, err := createTmpFile()
		if err != nil {
			return -1, err
		}

		// Open tmpfile
		r.tmpfile, err = os.OpenFile(fname, os.O_RDONLY, 0000)
		if err != nil {
			log.Warnf("Couldn't open tmpfile. [%v]", err.Error())
			return -1, err
		}

		// start download
		go func() {
			defer func() {
				r.downloadComplete = true
			}()

			if err := r.download(fname); err != nil {
				r.downloadErr = err
			}
		}()
	}

	n, err = r.tmpfile.ReadAt(p, off)
	if r.downloadComplete && err == io.EOF {
		log.Debugf("Sent EOF to the client. [%s]", r.sf.Name())
		return n, io.EOF

	} else if err == io.EOF {
		// wait for downloading
		//log.Debugf("Wait for downloading, offset=%d len=%d read=%d", off, len(p), n)
		return n, nil

	} else {
		// log.Debugf("ReadAt, offset=%d len=%d read=%d", off, len(p), n)
		return n, err
	}
}

func (r *swiftReader) Close() error {
	log.Debugf("swiftReader closed")
	return nil
}

type swiftWriter struct {
	swift *Swift
	sf    *SwiftFile

	tmpfile     *os.File
	contentSize int64
}

func (w *swiftWriter) WriteAt(p []byte, off int64) (n int, err error) {
	// if rw.tmpfile == nil {
	// 	fname := rw.sf.TempFileName()
	// 	log.Debugf("Create tmpfile to write. [%s]", fname)

	// 	// Do not need to call tmpfile.Close(). It'll be called in swiftReadWriter.Close()
	// 	rw.tmpfile, err = os.OpenFile(fname, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
	// 	if err != nil {
	// 		log.Warnf("%v", err.Error())
	// 		return 0, err
	// 	}
	// }

	// log.Debugf("Write to tmpfile, offset=%d len=%d", off, len(p))

	// // write buffer to the temporary file
	// _, err = rw.tmpfile.WriteAt(p, off)
	// if err != nil {
	// 	log.Warnf("%v", err.Error())
	// 	return 0, err
	// }

	// return len(p), nil
	return 0, nil
}

func (w *swiftWriter) Close() error {
	log.Debugf("swiftWriter closed")
	return nil
}

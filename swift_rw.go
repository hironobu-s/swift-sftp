package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"time"
)

// swiftReader implements io.ReadAt interface
type swiftReader struct {
	swift *Swift
	sf    *SwiftFile

	tmpfile          *os.File
	downloadComplete bool
	downloadErr      error
}

func (r *swiftReader) download(tmpFileName string) (err error) {
	log.Debugf("Create tmpfile. [%s]", tmpFileName)
	fw, err := os.OpenFile(tmpFileName, os.O_WRONLY|os.O_TRUNC, 0000)
	if err != nil {
		log.Warnf("%v", err.Error())
		return err
	}
	defer fw.Close()

	body, size, err := r.swift.Download(r.sf.Name())
	if err != nil {
		return err
	}
	defer body.Close()

	log.Debugf("Start downloading '%s' (size=%d) from Object Storage", r.sf.Name(), size)
	_, err = io.Copy(fw, body)
	if err != nil {
		log.Warnf("Error occured during copying [%v]", err.Error())
		return err
	}
	log.Debugf("Download completed")

	return nil
}

func (r *swiftReader) ReadAt(p []byte, off int64) (n int, err error) {
	if r.tmpfile == nil {
		log.Infof("Start sending '%s' (size=%d) to client", r.sf.Name(), r.sf.Size())

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
		log.Debugf("Send EOF to client. [%s]", r.sf.Name())
		return n, io.EOF

	} else if err == io.EOF {
		// wait for downloading
		return n, nil

	} else {
		return n, err
	}
}

func (r *swiftReader) Close() error {
	log.Infof("'%s' was sent successfully", r.sf.Name())
	return nil
}

// swiftWriter implements io.WriteAt interface
type swiftWriter struct {
	swift *Swift
	sf    *SwiftFile

	tmpfile        *os.File
	uploadComplete bool
	uploadErr      error
}

func (w *swiftWriter) upload() (err error) {
	fname := w.tmpfile.Name()
	log.Debugf("Upload: create tmpfile. [%s]", fname)
	fr, err := os.OpenFile(fname, os.O_RDONLY, 000)
	if err != nil {
		log.Warnf("%v", err.Error())
		return err
	}
	defer fr.Close()

	return w.swift.Put(w.sf.Name(), fr)
}

func (w *swiftWriter) WriteAt(p []byte, off int64) (n int, err error) {
	if w.tmpfile == nil {
		log.Infof("Start receiving '%s' from client", w.sf.Name())

		// Create tmpfile
		fname, err := createTmpFile()
		if err != nil {
			return -1, err
		}

		// Open tmpfile
		w.tmpfile, err = os.OpenFile(fname, os.O_WRONLY, 0000)
		if err != nil {
			log.Warnf("Couldn't open tmpfile. [%v]", err.Error())
			return -1, err
		}
	}

	n, err = w.tmpfile.WriteAt(p, off)
	if err != nil {
		log.Debugf("%v", err)
	}
	return n, err
}

func (w *swiftWriter) Close() error {
	// start uploading
	if w.tmpfile != nil {
		s, err := w.tmpfile.Stat()
		if err != nil {
			return err
		}

		log.Infof("Start uploading '%s' (size=%d) to Object Storage", w.sf.Name(), s.Size())

		//go func() {
		defer func() {
			w.uploadComplete = true
		}()

		if err := w.upload(); err != nil {
			w.uploadErr = err
			log.Debugf("Upload: complete with error. [%v]", err)
		}

		log.Infof("'%s' was uploaded successfully", w.sf.Name())

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

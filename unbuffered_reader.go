package main

import (
	"errors"
	"io"
	"io/ioutil"
)

// UnbufferedReader adds io.ReaderAt implementation to R (io.Reader).
type UnbufferedReader struct {
	R io.Reader
	n int64
}

func (u *UnbufferedReader) ReadAt(p []byte, offset int64) (n int, err error) {
	if offset < u.n {
		return 0, errors.New("invalid offset")
	}
	diff := offset - u.n
	written, err := io.CopyN(ioutil.Discard, u.R, diff)
	u.n += written
	if err != nil {
		return 0, err
	}

	n, err = u.R.Read(p)
	u.n += int64(n)
	return
}

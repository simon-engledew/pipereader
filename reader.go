// Package pipereader provides a reader that transforms its input through a writer without using a goroutine.
package pipereader

import (
	"bytes"
	"errors"
	"io"
)

type pipeReader struct {
	reader io.Reader
	writer io.Writer
	buf    bytes.Buffer
	drain  bool
}

func push(w io.Writer, r io.Reader, b []byte) error {
	nr, er := r.Read(b)
	if er != nil {
		return er
	}
	_, ew := w.Write(b[:nr])
	return ew
}

func (cr *pipeReader) Read(b []byte) (n int, err error) {
	for {
		n, err = cr.buf.Read(b)
		if !cr.drain && err == io.EOF {
			err = push(cr.writer, cr.reader, b)
			if errors.Is(err, io.EOF) {
				cr.drain = true
			}
			if err != nil {
				closeErr := cr.Close()
				if err == io.EOF {
					err = closeErr
				}
			}
			if err == nil {
				continue
			}
		}

		return n, err
	}
}

func (cr *pipeReader) Write(b []byte) (int, error) {
	return cr.buf.Write(b)
}

func (cr *pipeReader) Close() error {
	if wc, ok := cr.writer.(io.WriteCloser); cr.writer != cr && ok {
		return wc.Close()
	}
	return nil
}

// New returns a reader that will pass all the data read from r through the writer returned by fn.
func New[T io.Writer](r io.Reader, fn func(w io.Writer) T) io.ReadCloser {
	cr := &pipeReader{
		reader: r,
	}
	cr.writer = fn(cr)
	return cr
}

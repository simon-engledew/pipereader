package pipereader

import (
	"bytes"
	"errors"
	"io"
)

type pipeReader[T io.WriteCloser] struct {
	reader io.Reader
	writer T
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

func (cr *pipeReader[T]) Read(b []byte) (n int, err error) {
	for {
		n, err = cr.buf.Read(b)
		if !cr.drain && err == io.EOF {
			err = push(cr.writer, cr.reader, b)
			if errors.Is(err, io.EOF) {
				cr.drain = true
			}
			if err != nil {
				closeErr := cr.writer.Close()
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

func (cr *pipeReader[T]) Write(b []byte) (int, error) {
	return cr.buf.Write(b)
}

func (cr *pipeReader[T]) Close() error {
	return cr.writer.Close()
}

// New returns a reader that will pass all the data read from r through the writer returned by fn.
func New[T io.WriteCloser](r io.Reader, fn func(w io.Writer) T) io.ReadCloser {
	cr := &pipeReader[T]{
		reader: r,
	}
	cr.writer = fn(cr)
	return cr
}

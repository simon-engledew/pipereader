package pipereader_test

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/simon-engledew/pipereader"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

func FuzzMatches(f *testing.F) {
	testcases := []string{"Hello, world", " ", "!12345"}
	for _, tc := range testcases {
		f.Add(tc)
	}
	f.Fuzz(func(t *testing.T, v string) {
		actual := stream(t, pipereader.New(strings.NewReader(v), hex.Dumper))
		expected := buffer(t, []byte(v), hex.Dumper)

		assert(t, string(actual), string(expected))
	})
}

func assert(t testing.TB, actual, expected any) {
	t.Helper()
	if actual != expected {
		t.Errorf("Expected %+v got %+v", expected, actual)
	}
}

func stream(t testing.TB, r io.Reader) []byte {
	t.Helper()
	data, err := io.ReadAll(r)
	assert(t, err, nil)
	return data
}

func buffer[T io.WriteCloser](t testing.TB, data []byte, fn func(w io.Writer) T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := fn(&buf)

	_, err := io.Copy(w, bytes.NewReader(data))
	assert(t, err, nil)
	assert(t, w.Close(), nil)

	return buf.Bytes()
}

// fakeUpload is a stand-in for a method we cannot change
// it requires an io.Reader so we cannot just io.Copy to it.
func fakeUpload(r io.Reader) error {
	_, err := io.Copy(io.Discard, r)
	return err
}

func BenchmarkStream(b *testing.B) {
	r := io.LimitReader(rand.Reader, 1024*1024*10)

	b.Run("stream", func(b *testing.B) {
		assert(b, fakeUpload(pipereader.New(r, hex.Dumper)), nil)
	})

	b.Run("buffer", func(b *testing.B) {
		var buf bytes.Buffer

		w := hex.Dumper(&buf)
		_, err := io.Copy(w, r)
		assert(b, err, nil)
		assert(b, w.Close(), nil)
		assert(b, fakeUpload(&buf), nil)
	})
}

func TestDumper(t *testing.T) {
	data, err := io.ReadAll(io.LimitReader(rand.Reader, 1000))
	assert(t, err, nil)

	cr := pipereader.New(bytes.NewReader(data), hex.Dumper)
	actual := stream(t, cr)
	expected := buffer(t, data, hex.Dumper)

	assert(t, string(actual), string(expected))
}

func TestGzip(t *testing.T) {
	for _, size := range []int64{1000, 10000, 100000, 1000000} {
		t.Run(fmt.Sprintf("size-%d", size), func(t *testing.T) {
			data, err := io.ReadAll(io.LimitReader(rand.Reader, size))
			assert(t, err, nil)

			cr := pipereader.New(bytes.NewReader(data), gzip.NewWriter)

			actual := stream(t, cr)
			expected := buffer(t, data, gzip.NewWriter)

			assert(t, hex.EncodeToString(actual), hex.EncodeToString(expected))
		})
	}
}

var errClosed = errors.New("closed")

type errReader struct {
	io.Reader
	count int
}

func (e *errReader) Read(p []byte) (int, error) {
	e.count -= 1
	if e.count < 0 {
		return 0, errClosed
	}
	return e.Reader.Read(p)
}

func TestErrCloser(t *testing.T) {
	r := &errReader{Reader: rand.Reader, count: 10}

	_, err := io.Copy(io.Discard, pipereader.New(r, hex.Dumper))
	assert(t, err, errClosed)
}

func TestReader(t *testing.T) {
	cr := pipereader.New(strings.NewReader("content"), hex.Dumper)

	assert(t, iotest.TestReader(cr, []byte(hex.Dump([]byte("content")))), nil)
}

func TestHalfReader(t *testing.T) {
	r := iotest.HalfReader(io.LimitReader(rand.Reader, 1024*1024*10))

	cr := pipereader.New(r, bufio.NewWriter)

	var buf bytes.Buffer

	_, err := io.Copy(&buf, cr)
	assert(t, err, nil)
	assert(t, buf.Len(), 1024*1024*10)
}

func TestClose(t *testing.T) {
	r := io.LimitReader(rand.Reader, 1024*1024*10)

	cr := pipereader.New(r, hex.Dumper)
	assert(t, cr.Close(), nil)

	var buf bytes.Buffer

	_, err := io.Copy(&buf, cr)
	assert(t, err.Error(), "encoding/hex: dumper closed")
}

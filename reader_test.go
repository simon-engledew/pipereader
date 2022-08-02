package pipereader

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func FuzzMatches(f *testing.F) {
	testcases := []string{"Hello, world", " ", "!12345"}
	for _, tc := range testcases {
		f.Add(tc) // Use f.Add to provide a seed corpus
	}
	f.Fuzz(func(t *testing.T, v string) {
		actual := stream(t, New(strings.NewReader(v), hex.Dumper))
		expected := buffer(t, []byte(v), hex.Dumper)

		if bytes.Compare(expected, actual) != 0 {
			t.Errorf("buffers do not match:\n%q\n!=\n%q", string(expected), string(actual))
		}
	})
}

func stream(t testing.TB, r io.Reader) []byte {
	t.Helper()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Error(err.Error())
	}
	return data
}

func buffer[T io.WriteCloser](t testing.TB, data []byte, fn func(w io.Writer) T) []byte {
	var buf bytes.Buffer
	w := fn(&buf)

	_, err := io.Copy(w, bytes.NewReader(data))
	if err != nil {
		t.Error(err.Error())
	}

	err = w.Close()
	if err != nil {
		t.Error(err.Error())
	}
	return buf.Bytes()
}

// fakeUpload is a stand-in for a method we cannot change
// it requires an io.Reader so we cannot just io.Copy to it.
func fakeUpload(r io.Reader) error {
	_, err := io.Copy(os.Stdout, r)
	return err
}

func BenchmarkStream(b *testing.B) {
	r := io.LimitReader(rand.Reader, 1024*1024*10)

	b.Run("stream", func(b *testing.B) {
		fakeUpload(New(r, hex.Dumper))
	})

	b.Run("buffer", func(b *testing.B) {
		var buf bytes.Buffer

		w := hex.Dumper(&buf)
		io.Copy(w, r)
		w.Close()

		fakeUpload(&buf)
	})
}

func TestDumper(t *testing.T) {
	data, err := io.ReadAll(io.LimitReader(rand.Reader, 1000))
	if err != nil {
		t.Error(err.Error())
	}

	cr := New(bytes.NewReader(data), hex.Dumper)
	actual := stream(t, cr)
	expected := buffer(t, data, hex.Dumper)

	if bytes.Compare(expected, actual) != 0 {
		t.Errorf("buffers do not match:\n%q\n!=\n%q", string(expected), string(actual))
	}
}

func TestGzip(t *testing.T) {
	for _, size := range []int64{1000, 10000, 100000, 1000000} {
		t.Run(fmt.Sprintf("size-%d", size), func(t *testing.T) {
			data, err := io.ReadAll(io.LimitReader(rand.Reader, size))
			if err != nil {
				t.Error(err.Error())
			}

			cr := New(bytes.NewReader(data), gzip.NewWriter)

			actual := stream(t, cr)
			expected := buffer(t, data, gzip.NewWriter)

			if bytes.Compare(expected, actual) != 0 {
				t.Errorf("buffers do not match:\n%q\n!=\n%q", hex.EncodeToString(expected), hex.EncodeToString(actual))
			}
		})
	}
}

func TestClose(t *testing.T) {
	r := io.LimitReader(rand.Reader, 1024*1024*10)

	cr := New(r, hex.Dumper)
	err := cr.Close()
	if err != nil {
		t.Error(err.Error())
	}

	var buf bytes.Buffer

	_, err = io.Copy(&buf, cr)
	if err == nil {
		t.Error("error expected")
	}
}

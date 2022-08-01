package pipereader

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"runtime"
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

func BenchmarkStream(b *testing.B) {
	r := io.LimitReader(rand.Reader, 1024*1024*10)

	cr := New(r, gzip.NewWriter)

	b.Run("stream", func(b *testing.B) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		before := m.TotalAlloc

		io.Copy(io.Discard, cr)

		runtime.ReadMemStats(&m)
		fmt.Println(m.TotalAlloc - before)
	})

	b.Run("buffer", func(b *testing.B) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		before := m.TotalAlloc

		w := gzip.NewWriter(io.Discard)
		io.Copy(w, r)
		w.Close()

		runtime.ReadMemStats(&m)
		fmt.Println(m.TotalAlloc - before)
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

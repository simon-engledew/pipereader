package pipereader_test

import (
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
		actual, err := io.ReadAll(pipereader.New(strings.NewReader(v), hex.Dumper))
		assertNil(t, err)
		expected := hex.Dump([]byte(v))
		assert(t, string(actual), expected)
	})
}

func randomData(t testing.TB, n int64) []byte {
	t.Helper()
	data, err := io.ReadAll(io.LimitReader(rand.Reader, n))
	assertNil(t, err)
	return data
}

func assert[T comparable](t testing.TB, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Fatalf("Expected:\n%+v\nGot:\n%+v", expected, actual)
	}
}

func assertNil(t testing.TB, actual any) {
	t.Helper()
	if actual != nil {
		t.Errorf("Expected %+v got %+v", nil, actual)
	}
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
		assertNil(b, fakeUpload(pipereader.New(r, hex.Dumper)))
	})

	b.Run("buffer", func(b *testing.B) {
		var buf bytes.Buffer

		w := hex.Dumper(&buf)
		_, err := io.Copy(w, r)
		assertNil(b, err)
		assertNil(b, w.Close())
		assertNil(b, fakeUpload(&buf))
	})
}

func TestDumper(t *testing.T) {
	data := randomData(t, 1000)
	cr := pipereader.New(bytes.NewReader(data), hex.Dumper)
	actual, err := io.ReadAll(cr)
	assertNil(t, err)
	assert(t, string(actual), hex.Dump(data))
}

func TestGzip(t *testing.T) {
	for _, size := range []int64{1000, 10000, 100000, 1000000} {
		t.Run(fmt.Sprintf("size-%d", size), func(t *testing.T) {
			data := randomData(t, size)

			cr := pipereader.New(bytes.NewReader(data), gzip.NewWriter)

			actual, err := io.ReadAll(cr)
			assertNil(t, err)

			var buf bytes.Buffer
			w := gzip.NewWriter(&buf)

			_, err = io.Copy(w, bytes.NewReader(data))
			assertNil(t, err)
			assertNil(t, w.Close())

			expected := buf.Bytes()

			assert(t, hex.EncodeToString(actual), hex.EncodeToString(expected))
		})
	}
}

var errClosed = errors.New("closed")

func TestErrCloser(t *testing.T) {
	r := io.MultiReader(io.LimitReader(rand.Reader, 10), iotest.ErrReader(errClosed))

	_, err := io.Copy(io.Discard, pipereader.New(r, identity))
	assert(t, errors.Is(err, errClosed), true)
}

func TestReader(t *testing.T) {
	data := randomData(t, 1024*16)

	cr := pipereader.New(bytes.NewReader(data), identity)

	assertNil(t, iotest.TestReader(cr, data))
}

func identity(w io.Writer) io.Writer {
	return w
}

func TestHalfReader(t *testing.T) {
	data := randomData(t, 1024*16)

	cr := pipereader.New(iotest.HalfReader(bytes.NewReader(data)), identity)

	actual, err := io.ReadAll(cr)
	assertNil(t, err)
	assert(t, len(actual), len(data))
	assert(t, bytes.Compare(actual, data), 0)
}

func TestClose(t *testing.T) {
	r := io.LimitReader(rand.Reader, 1024*16)

	cr := pipereader.New(r, hex.Dumper)
	assertNil(t, cr.Close())

	var buf bytes.Buffer

	n, err := io.Copy(&buf, cr)
	assert(t, err != nil, true)
	assert(t, err.Error(), "encoding/hex: dumper closed")
	assert(t, n, 0)
}

# pipereader

A go library for streaming readers through writers without using `io.Pipe` or a goroutine.

e.g: compressing/zipping readers:

```go
package main

import (
	"io"
	"os"
	"compress/gzip"
	"github.com/simon-engledew/pipereader"
)

func main() {
	io.Copy(os.Stdout, pipereader.New(os.Stdin, gzip.NewWriter))
}
```

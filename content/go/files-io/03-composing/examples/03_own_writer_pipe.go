package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

// countingWriter is a complete io.Writer. Anything with this one method plugs
// into io.Copy, io.MultiWriter, fmt.Fprintf, json.NewEncoder, gzip.NewWriter...
type countingWriter struct {
	bytes, writes int
}

func (c *countingWriter) Write(p []byte) (int, error) {
	c.bytes += len(p)
	c.writes++
	return len(p), nil
}

type order struct {
	ID   int    `json:"id"`
	City string `json:"city"`
}

func main() {
	pr, pw := io.Pipe() // no buffer: each Write waits for a Read to take it

	go func() {
		enc := json.NewEncoder(pw) // the producer only sees an io.Writer
		var err error
		for i, city := range []string{"Bursa", "Muğla", "Çorum"} {
			if err = enc.Encode(order{ID: i + 1, City: city}); err != nil {
				break
			}
		}
		pw.CloseWithError(err) // nil means EOF for the reader; otherwise it gets err
	}()

	var count countingWriter
	if _, err := io.Copy(io.MultiWriter(os.Stdout, &count), pr); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d bytes in %d writes, one record at a time\n", count.bytes, count.writes)
}

package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
)

func writeGzip(name string, rows int) (err error) {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()

	zw := gzip.NewWriter(f) // an io.Writer that compresses into another io.Writer
	zw.Name = "siparisler.csv"
	for i := range rows {
		fmt.Fprintf(zw, "%d;İstanbul;%d\n", i, i*7)
	}
	return zw.Close() // writes the footer (CRC-32 and size); f.Close doesn't
}

func main() {
	if err := writeGzip("siparisler.csv.gz", 10000); err != nil {
		log.Fatal(err)
	}
	f, err := os.Open("siparisler.csv.gz")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f) // an io.Reader that decompresses
	if err != nil {
		log.Fatal(err)
	}
	sc := bufio.NewScanner(zr)
	lines, last := 0, ""
	for sc.Scan() {
		lines, last = lines+1, sc.Text()
	}
	if err := sc.Err(); err != nil { // a corrupt or truncated file shows up here
		log.Fatal(err)
	}
	fmt.Println(zr.Name, "lines:", lines, "last:", last)

	bad := gzip.NewWriter(io.Discard)
	bad.Name = "Şubat.csv" // header strings are Latin-1 only; Ş is U+015E
	_, err = bad.Write([]byte("x"))
	fmt.Println(err)

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Write([]byte("ay;tutar\nKasım;100\n"))
	zw.Flush() // Flush without Close: the data is there, the footer is not
	zr2, _ := gzip.NewReader(&buf)
	data, err := io.ReadAll(zr2)
	fmt.Printf("%q, %v\n", data, err)
}

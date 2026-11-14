package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const export = `id;city;amount
1;İstanbul;120,50
2;Şanlıurfa;75,00
3;Iğdır;9,90
`

func main() {
	// One pass, two destinations: the file and a running SHA-256.
	dst, err := os.Create("export.csv")
	if err != nil {
		log.Fatal(err)
	}
	sent := sha256.New() // a hash.Hash is an io.Writer
	n, err := io.Copy(io.MultiWriter(dst, sent), strings.NewReader(export))
	if err != nil {
		log.Fatal(err)
	}
	if err := dst.Close(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("copied %d bytes for %d characters\n", n, len([]rune(export)))

	// Read once, consume twice: the scanner parses, TeeReader feeds the hash.
	src, err := os.Open("export.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer src.Close()
	got := sha256.New()
	sc := bufio.NewScanner(io.TeeReader(src, got))
	rows := 0
	for sc.Scan() {
		rows++
	}
	if err := sc.Err(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("lines read:", rows)
	fmt.Printf("sha256 %x…, same as sent: %t\n", got.Sum(nil)[:8], bytes.Equal(got.Sum(nil), sent.Sum(nil)))
}

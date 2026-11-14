package main

import (
	"archive/zip"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

func main() {
	f, err := os.Create("rapor.zip")
	if err != nil {
		log.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, name := range []string{"raporlar/Ocak.csv", "raporlar/Şubat.csv", "../../evil.txt"} {
		w, err := zw.Create(name) // Deflate; the writer doesn't validate the name
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(w, "ay;tutar\n%s;100\n", filepath.Base(name))
	}
	if err := zw.Close(); err != nil { // writes the central directory: no Close, no archive
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}

	zr, err := zip.OpenReader("rapor.zip")
	if err != nil {
		log.Fatal(err)
	}
	defer zr.Close()
	for _, zf := range zr.File { // raw entries: names exactly as stored
		fmt.Printf("%-20q utf8-flag=%-5t IsLocal=%t\n",
			zf.Name, zf.Flags&0x800 != 0, filepath.IsLocal(zf.Name))
	}

	// The fs.FS view cleans names: no "..", no leading "/". CopyFS (Go 1.23)
	// extracts through it, so nothing lands outside "out".
	if err := os.CopyFS("out", zr); err != nil {
		log.Fatal(err)
	}
	err = fs.WalkDir(os.DirFS("out"), ".", func(name string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			fmt.Println("extracted:", name)
		}
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
}

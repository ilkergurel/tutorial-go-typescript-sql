package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

func main() {
	// Build a small tree to look at. os functions accept "/" on Windows too.
	for _, name := range []string{
		"data/İzmir.txt", "data/Zonguldak.txt", "data/notlar.md",
		"data/raporlar/Ocak.csv", "data/raporlar/Şubat.csv",
		"data/node_modules/lib/index.js",
	} {
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil { // mkdir -p
			log.Fatal(err)
		}
		if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
			log.Fatal(err)
		}
	}

	entries, err := os.ReadDir("data") // one level, sorted by name as bytes
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range entries {
		fmt.Printf("dir=%-5t %s\n", e.IsDir(), e.Name())
	}

	err = filepath.WalkDir("data", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err // an unreadable directory: stop, or log it and return nil
		}
		if d.IsDir() && d.Name() == "node_modules" {
			return fs.SkipDir // don't descend
		}
		if filepath.Ext(path) == ".csv" {
			fmt.Println("walk:", filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// The io/fs view of the same tree: relative, always "/"-separated names.
	matches, err := fs.Glob(os.DirFS("data"), "raporlar/*.csv")
	fmt.Println("glob:", matches, err)
}

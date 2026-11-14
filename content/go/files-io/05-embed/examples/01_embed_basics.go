package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
)

//go:embed version.txt
var version string // the file's content, read by the compiler

//go:embed static
var static embed.FS // a directory: names starting with . or _ are left out

//go:embed all:static
var everything embed.FS // all: keeps them, .env.local included

func list(label string, fsys fs.FS) {
	fmt.Println(label)
	err := fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			fmt.Println("  " + name)
		}
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	fmt.Printf("version: %q\n", version)
	if err := os.WriteFile("version.txt", []byte("9.9.9\n"), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("after rewriting version.txt on disk: %q\n", version)

	list("//go:embed static", static)
	list("//go:embed all:static", everything)

	page, err := static.ReadFile("static/index.html") // the directory stays in the name
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("index.html starts with %q\n", page[:15])
	_, err = static.Open("static/_draft.html")
	fmt.Println("_draft.html not embedded:", err)
}

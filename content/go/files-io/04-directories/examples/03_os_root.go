package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	// A private file next to the upload directory.
	if err := os.WriteFile("secret.env", []byte("DB_PASSWORD=hunter2"), 0o600); err != nil {
		log.Fatal(err)
	}
	if err := os.Mkdir("uploads", 0o755); err != nil {
		log.Fatal(err)
	}

	root, err := os.OpenRoot("uploads") // every name below resolves inside uploads/
	if err != nil {
		log.Fatal(err)
	}
	defer root.Close()

	// MkdirAll, WriteFile and ReadFile on Root arrived in Go 1.25.
	if err := root.MkdirAll("2026/kasım", 0o755); err != nil {
		log.Fatal(err)
	}
	if err := root.WriteFile("2026/kasım/fatura.txt", []byte("tutar=1.250,50"), 0o644); err != nil {
		log.Fatal(err)
	}

	// Names as they might arrive in an HTTP request.
	for _, name := range []string{"2026/kasım/fatura.txt", "../secret.env", "2026/../../secret.env"} {
		naive, _ := os.ReadFile(filepath.Join("uploads", name))
		safe, err := root.ReadFile(name)
		fmt.Printf("%q IsLocal=%t\n", name, filepath.IsLocal(name))
		fmt.Printf("  os.ReadFile(Join): %q\n", naive)
		fmt.Printf("  root.ReadFile:     %q %v\n", safe, err)
	}
}

package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	data, err := os.ReadFile("app.log") // whole file -> []byte
	if err != nil {
		log.Fatalf("read app.log: %v", err)
	}

	text := string(data)
	lines := strings.Split(strings.TrimSpace(text), "\n")

	fmt.Printf("read %d bytes, %d lines\n", len(data), len(lines))
	fmt.Println("first line:", lines[0])
}

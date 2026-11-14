package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// The package the compiler will explain, one decision per function.
const src = `package esc

import "fmt"

type Point struct{ X, Y int }

func NewPoint(x, y int) *Point {
	p := Point{x, y}
	return &p
}

func Sum() int {
	p := &Point{1, 2}
	return p.X + p.Y
}

func Keep(n int) []byte { return make([]byte, n) }

func Scratch(n int) int {
	buf := make([]byte, n)
	return len(buf)
}

func Huge() byte {
	var buf [1 << 20]byte
	return buf[0]
}

func Label(p Point) string { return fmt.Sprint(p.X) }
`

func main() {
	dir, err := os.MkdirTemp("", "esc-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	write(filepath.Join(dir, "go.mod"), "module example.com/esc\n\ngo 1.27\n")
	write(filepath.Join(dir, "esc.go"), src)

	cmd := exec.Command("go", "build", "-gcflags=-m", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Sprint(err, string(out)))
	}
	fmt.Println("$ go build -gcflags=-m .")
	for _, line := range strings.Split(filepath.ToSlash(string(out)), "\n") {
		if !strings.Contains(line, "can inline") && strings.HasPrefix(line, "./") {
			fmt.Println(line)
		}
	}
}

func write(path, body string) {
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		panic(err)
	}
}

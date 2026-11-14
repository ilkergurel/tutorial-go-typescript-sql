package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Code written for Go 1.21 and older, in a module that says go 1.21.
var files = map[string]string{
	"go.mod": "module example.com/old\n\ngo 1.21\n",
	"old.go": `package old

import "fmt"

func Show(v interface{}) { fmt.Println(v) }

func Smaller(a, b int) int {
	m := a
	if b < m {
		m = b
	}
	return m
}

func Squares(n int) (out []int) {
	for i := 0; i < n; i++ {
		out = append(out, i*i)
	}
	return out
}

func PrintAll(names []string) {
	for _, n := range names {
		n := n
		defer func() { fmt.Println(n) }()
	}
}
`,
}

func main() {
	dir, err := os.MkdirTemp("", "old-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			panic(err)
		}
	}

	gocmd(dir, "fix", "-diff", ".") // preview only
	gocmd(dir, "fix", ".")          // apply
	gocmd(dir, "get", "go@1.22.0")  // raise the module's Go version
	gocmd(dir, "fix", "-diff", ".")
}

// gocmd runs the go command offline in dir. The diff's ---/+++ header lines
// hold absolute temp paths, so they are dropped.
func gocmd(dir string, args ...string) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off")
	out, err := cmd.CombinedOutput()
	fmt.Printf("$ go %s\n", strings.Join(args, " "))
	for line := range strings.Lines(string(out)) {
		if !strings.HasPrefix(line, "--- ") && !strings.HasPrefix(line, "+++ ") {
			fmt.Print(line)
		}
	}
	if err != nil {
		fmt.Printf("(%v)\n", err)
	}
}

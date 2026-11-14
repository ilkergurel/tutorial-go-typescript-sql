package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var files = map[string]string{
	"go.mod": "module example.com/greet\n\ngo 1.27\n",
	// Compiles, but %d gets a string and the indentation isn't gofmt's.
	"greet.go": `package greet

import "fmt"

func Hello(name string) string {
    return fmt.Sprintf("Merhaba, %d", name)
}
`,
	"greet_test.go": `package greet

import "testing"

func TestHello(t *testing.T) {
	if got := Hello("Ayşe"); got != "Merhaba, Ayşe" {
		t.Errorf("got %q", got)
	}
}
`,
}

func main() {
	dir, err := os.MkdirTemp("", "greet-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			panic(err)
		}
	}

	gocmd(dir, "build", "./...")
	gocmd(dir, "vet", "./...")
	gocmd(dir, "test", "./...")
	gocmd(dir, "test", "-vet=off", "./...")
	gocmd(dir, "fmt", "./...")
}

var timing = regexp.MustCompile(`\s+\(?[0-9]+\.[0-9]+s\)?`)

// gocmd runs the go command offline in dir. Output is normalized so it reads
// the same on every OS and every run: "/" in paths, test timings removed.
func gocmd(dir string, args ...string) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off")
	out, err := cmd.CombinedOutput()
	text := timing.ReplaceAllString(filepath.ToSlash(string(out)), "")
	fmt.Printf("$ go %s\n%s", strings.Join(args, " "), text)
	if err != nil {
		fmt.Println("->", err)
	}
}

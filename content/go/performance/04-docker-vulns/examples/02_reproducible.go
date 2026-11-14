package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const app = `package main

import "fmt"

func main() { fmt.Println("release candidate") }
`

func main() {
	root, err := os.MkdirTemp("", "repro-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)

	// The same commit checked out twice: on a laptop and on a CI runner.
	laptop := checkout(filepath.Join(root, "home", "ayse", "src", "svc"))
	ci := checkout(filepath.Join(root, "builds", "1234", "svc"))

	for _, flags := range [][]string{nil, {"-trimpath"}} {
		a, b := build(laptop, flags), build(ci, flags)
		label := "go build"
		if flags != nil {
			label += " " + flags[0]
		}
		fmt.Printf("%-18s identical: %-5v  laptop path inside: %v\n", label,
			sha256.Sum256(a) == sha256.Sum256(b), bytes.Contains(a, []byte(filepath.ToSlash(laptop))))
	}
}

func checkout(dir string) string {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	write(filepath.Join(dir, "go.mod"), "module example.com/svc\n\ngo 1.27\n")
	write(filepath.Join(dir, "main.go"), app)
	return dir
}

// build produces the linux/amd64 release binary and returns its bytes.
func build(dir string, flags []string) []byte {
	out := filepath.Join(dir, "svc.bin")
	args := append(append([]string{"build"}, flags...), "-o", out, ".")
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off",
		"GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	if msg, err := cmd.CombinedOutput(); err != nil {
		panic(fmt.Sprint(err, string(msg)))
	}
	b, err := os.ReadFile(out)
	if err != nil {
		panic(err)
	}
	return b
}

func write(path, body string) {
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		panic(err)
	}
}

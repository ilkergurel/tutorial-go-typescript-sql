package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const mainGo = `package main

import "fmt"

func main() {
	for i := range 3 { // range over an int: Go 1.22
		fmt.Print(i)
	}
	fmt.Println()
}
`

func main() {
	dir, err := os.MkdirTemp("", "golines-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	write(dir, "main.go", mainGo)

	// The go line is the language version: newer syntax is a compile error.
	write(dir, "go.mod", "module example.com/tc\n\ngo 1.21\n")
	gocmd(dir, "local", "run", ".")

	write(dir, "go.mod", "module example.com/tc\n\ngo 1.22\n")
	gocmd(dir, "local", "run", ".")

	// It is also the minimum toolchain. local refuses; auto (the default)
	// tries to download the toolchain go.mod asks for.
	write(dir, "go.mod", "module example.com/tc\n\ngo 1.99\n")
	gocmd(dir, "local", "run", ".")
	gocmd(dir, "auto", "run", ".")
}

func write(dir, name, body string) {
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		panic(err)
	}
}

// gocmd runs the go command in dir with the given GOTOOLCHAIN. GOPROXY=off
// makes any download fail at once instead of reaching the network.
func gocmd(dir, toolchain string, args ...string) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN="+toolchain, "GOPROXY=off")
	out, err := cmd.CombinedOutput()
	fmt.Printf("$ GOTOOLCHAIN=%s go %s\n%s", toolchain, strings.Join(args, " "), filepath.ToSlash(string(out)))
	if err != nil {
		fmt.Printf("(%v)\n", err)
	}
}

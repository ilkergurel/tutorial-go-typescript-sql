package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Two packages (two directories). go mod init below makes them one module.
var files = map[string]string{
	"main.go": `package main

import (
	"fmt"

	"example.com/hello/greet"
)

func main() { fmt.Println(greet.Hello("Ayşe")) }
`,
	"greet/greet.go": `// Package greet builds greetings.
package greet

// Hello greets name in Turkish.
func Hello(name string) string { return "Merhaba, " + name }
`,
}

func main() {
	dir, err := os.MkdirTemp("", "hello-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			panic(err)
		}
	}

	gocmd(dir, "mod", "init", "example.com/hello")
	gocmd(dir, "list", "-m")
	gocmd(dir, "list", "-f", "{{.ImportPath}} is package {{.Name}}", "./...")
	gocmd(dir, "run", ".")
	gocmd(dir, "doc", "./greet")
}

// gocmd runs the go command in dir and prints what a terminal would show.
// GOTOOLCHAIN=local and GOPROXY=off: the demo never downloads anything.
func gocmd(dir string, args ...string) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off")
	out, err := cmd.CombinedOutput()

	shown := []string{"$ go"}
	for _, a := range args {
		if strings.Contains(a, " ") {
			a = strconv.Quote(a)
		}
		shown = append(shown, a)
	}
	fmt.Printf("%s\n%s", strings.Join(shown, " "), out)
	if err != nil {
		fmt.Println("->", err)
	}
}

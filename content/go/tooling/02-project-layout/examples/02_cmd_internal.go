package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// The usual shape of an application module: binaries under cmd/, the code
// they share under internal/.
var files = map[string]string{
	"go.mod": "module example.com/shop\n\ngo 1.27\n",
	"cmd/shopd/main.go": `package main

import (
	"fmt"

	"example.com/shop/internal/vat"
)

func main() { fmt.Println("gross:", 1000+vat.Of(1000)) }
`,
	"cmd/peek/main.go": `package main

import (
	"fmt"

	"example.com/shop/internal/vat"
)

func main() { fmt.Println(vat.round(1999, 100)) }
`,
	"internal/vat/vat.go": `package vat

// Of returns the 20% VAT on net, in kuruş.
func Of(net int) int { return round(net*20, 100) }

// round is unexported: only package vat can call it.
func round(a, b int) int { return (a + b/2) / b }
`,
}

func main() {
	for name, body := range files { // the runner starts in a fresh directory
		path := filepath.Join("shop", name)
		os.MkdirAll(filepath.Dir(path), 0o755) // if this fails, WriteFile fails too
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			panic(err)
		}
	}
	gocmd("list", "./...")
	gocmd("run", "./cmd/shopd")
	gocmd("build", "./...")
}

// gocmd runs the go command offline in shop; "/" in paths on every OS.
func gocmd(args ...string) {
	cmd := exec.Command("go", args...)
	cmd.Dir = "shop"
	cmd.Env = append(os.Environ(), "GOPROXY=off")
	out, err := cmd.CombinedOutput()
	fmt.Printf("$ go %s\n%s", strings.Join(args, " "), filepath.ToSlash(string(out)))
	if err != nil {
		fmt.Println("->", err)
	}
}

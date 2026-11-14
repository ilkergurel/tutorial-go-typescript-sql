package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Two modules side by side, as when you change a library and the app that
// uses it together. app/go.mod does not require example.com/shop.
var files = map[string]string{
	"shop/go.mod": "module example.com/shop\n\ngo 1.27\n",
	"shop/shop.go": `package shop

import "example.com/shop/internal/vat"

func Gross(net int) int { return net + vat.Of(net) }
`,
	"shop/internal/vat/vat.go": `package vat

func Of(net int) int { return net * 20 / 100 }
`,
	"app/go.mod": "module example.com/app\n\ngo 1.27\n",
	"app/main.go": `package main

import (
	"fmt"

	"example.com/shop"
)

func main() { fmt.Println("gross:", shop.Gross(1000)) }
`,
	"app/sneaky/sneaky.go": `package sneaky

import "example.com/shop/internal/vat"

var Rate = vat.Of(100)
`,
}

// go.work lists the module directories to treat as one build.
const goWork = `go 1.27

use (
	./app
	./shop
)
`

func main() {
	for name, body := range files { // the runner starts in a fresh directory
		os.MkdirAll(filepath.Dir(name), 0o755) // if this fails, WriteFile fails too
		if err := os.WriteFile(name, []byte(body), 0o644); err != nil {
			panic(err)
		}
	}
	gocmd("run", ".") // no workspace: example.com/shop is unknown

	// go.work in a parent directory turns both modules into main modules.
	if err := os.WriteFile("go.work", []byte(goWork), 0o644); err != nil {
		panic(err)
	}
	gocmd("run", ".")
	gocmd("build", "./...") // sneaky imports shop's internal package
}

// gocmd runs the go command offline in app; "/" in paths on every OS.
func gocmd(args ...string) {
	cmd := exec.Command("go", args...)
	cmd.Dir = "app"
	cmd.Env = append(os.Environ(), "GOPROXY=off")
	out, err := cmd.CombinedOutput()
	fmt.Printf("$ go %s\n%s", strings.Join(args, " "), filepath.ToSlash(string(out)))
	if err != nil {
		fmt.Println("->", err)
	}
}

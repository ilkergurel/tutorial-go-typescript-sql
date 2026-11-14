package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Before: order borrows util.Slug, then util grows a helper that takes an Order.
var before = map[string]string{
	"go.mod": "module example.com/shop\n\ngo 1.27\n",
	"order/order.go": `package order

import "example.com/shop/util"

type Order struct{ ID string }

func New(customer string) Order { return Order{ID: util.Slug(customer)} }
`,
	"util/util.go": `package util

import (
	"strings"

	"example.com/shop/order"
)

func Slug(s string) string { return strings.ToLower(strings.ReplaceAll(s, " ", "-")) }

func OrderURL(o order.Order) string { return "/orders/" + o.ID }
`,
}

// After: no util. Slug has a package named for its one job, and the URL
// helper is a method of the type it works on.
var after = map[string]string{
	"go.mod": "module example.com/shop\n\ngo 1.27\n",
	"slug/slug.go": `package slug

import "strings"

func Make(s string) string { return strings.ToLower(strings.ReplaceAll(s, " ", "-")) }
`,
	"order/order.go": `package order

import "example.com/shop/slug"

type Order struct{ ID string }

func New(customer string) Order { return Order{ID: slug.Make(customer)} }

func (o Order) URL() string { return "/orders/" + o.ID }
`,
}

func main() {
	build("before", before)
	build("after", after)
}

// build writes the module into dir and runs go build there. The runner
// starts every run in a fresh directory, so nothing needs cleaning up.
func build(dir string, files map[string]string) {
	for name, body := range files {
		path := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(path), 0o755) // if this fails, WriteFile fails too
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			panic(err)
		}
	}
	fmt.Println(dir + "$ go build ./...")
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir, cmd.Stdout, cmd.Stderr = dir, os.Stdout, os.Stdout
	fmt.Println("->", cmd.Run())
}

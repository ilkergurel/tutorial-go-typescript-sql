package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Placing an order sends an email, and the email needs the order's fields.
// If order imported email, the two packages would import each other. So
// order declares what it needs, email provides it, and main connects them.
var shop = map[string]string{
	"go.mod": "module example.com/shop\n\ngo 1.27\n",
	"order/order.go": `package order

type Order struct{ ID, Customer string }

// Notifier is declared by its consumer. Any type with this method fits.
type Notifier interface{ OrderPlaced(o Order) }

func Place(o Order, n Notifier) { n.OrderPlaced(o) }
`,
	"email/email.go": `package email

import (
	"fmt"

	"example.com/shop/order"
)

type Sender struct{ From string }

func (s Sender) OrderPlaced(o order.Order) {
	fmt.Printf("%s to %s: order %s confirmed\n", s.From, o.Customer, o.ID)
}
`,
	"main.go": `package main

import (
	"example.com/shop/email"
	"example.com/shop/order"
)

func main() {
	o := order.Order{ID: "A-1", Customer: "ayse@example.com"}
	order.Place(o, email.Sender{From: "shop@example.com"})
}
`,
}

const graph = "{{.ImportPath}} -> {{.Imports}}" // each package and what it imports

func main() {
	for name, body := range shop { // the runner starts in a fresh directory
		path := filepath.Join("shop", name)
		os.MkdirAll(filepath.Dir(path), 0o755) // if this fails, WriteFile fails too
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			panic(err)
		}
	}
	fmt.Printf("$ go list -f %q ./...\n", graph)
	gocmd("list", "-f", graph, "./...")
	fmt.Println("$ go run .")
	gocmd("run", ".")
}

// gocmd runs the go command in shop. Its stderr goes to stdout too, because
// stdout is what the lesson's check compares.
func gocmd(args ...string) {
	cmd := exec.Command("go", args...)
	cmd.Dir, cmd.Stdout, cmd.Stderr = "shop", os.Stdout, os.Stdout
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

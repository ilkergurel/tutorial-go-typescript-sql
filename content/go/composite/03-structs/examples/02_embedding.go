package main

import "fmt"

// Audit is meant to be embedded: it adds fields and methods to any struct.
type Audit struct {
	By      string
	Version int
}

func (a *Audit) Touch(by string) {
	a.By = by
	a.Version++
}

func (a Audit) Label() string { return "audit" }

// Summary calls Label on its own receiver, an Audit. Nothing is virtual.
func (a Audit) Summary() string {
	return fmt.Sprintf("%s v%d by %s", a.Label(), a.Version, a.By)
}

type Order struct {
	Audit // embedded: no field name, the type name is the field name
	ID    int
	Total int // kuruş
}

// Label on Order shadows the promoted Audit.Label.
func (o Order) Label() string { return fmt.Sprintf("order %d", o.ID) }

func main() {
	// Go 1.27: promoted fields (By, Version) may be keys in the literal.
	o := Order{ID: 7, Total: 1250, By: "ayşe", Version: 1}

	o.Touch("mehmet") // promoted method: means o.Audit.Touch("mehmet")
	fmt.Println(o.Version, o.By, o.Audit.By)

	fmt.Println(o.Label())   // Order's own method
	fmt.Println(o.Summary()) // Audit's method, which calls Audit.Label
	fmt.Printf("%+v\n", o)
}

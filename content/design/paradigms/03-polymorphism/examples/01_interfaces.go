package main

import (
	"encoding/json"
	"fmt"
)

// The behavior every variant shares. Any type with this method is a Discount,
// in this package or any other: the set of variants stays open.
type Discount interface {
	Apply(priceKurus int64) int64
}

type Percent struct{ Pct int64 }

func (d Percent) Apply(p int64) int64 { return p - p*d.Pct/100 }

type Fixed struct{ Kurus int64 }

func (d Fixed) Apply(p int64) int64 { return max(p-d.Kurus, 0) }

// A variant added later, without touching Percent, Fixed or main's loop.
type Chain []Discount

func (c Chain) Apply(p int64) int64 {
	for _, d := range c {
		p = d.Apply(p)
	}
	return p
}

// An operation the interface lacks needs a type switch, and the compiler
// doesn't check that the switch covers every variant.
func kind(d Discount) string {
	switch d.(type) {
	case Percent:
		return "percent"
	case Fixed:
		return "fixed"
	}
	return fmt.Sprintf("unknown %T", d)
}

func main() {
	const price = 20000 // 200,00 TL in kuruş
	offers := []struct {
		code string
		d    Discount
	}{
		{"DEV250", Fixed{25000}},
		{"HOSGELDIN", Fixed{5000}},
		{"KASIM15", Percent{15}},
		{"COMBO", Chain{Percent{10}, Fixed{1000}}},
	}
	for _, o := range offers {
		b, _ := json.Marshal(o.d) // can't fail for these types
		fmt.Printf("%-9s %5d  %-18s %s\n", o.code, o.d.Apply(price), kind(o.d), b)
	}
}

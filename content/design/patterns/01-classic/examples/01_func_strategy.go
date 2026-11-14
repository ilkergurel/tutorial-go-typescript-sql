// LEARN(design.patterns.classic): strategy, decorator and adapter as one func type and plain wrappers.
package main

import (
	"fmt"
	"math"
)

type Order struct {
	Desi  float64 // volumetric weight: how Turkish carriers price a parcel
	Kurus int     // order total; 100 kuruş = 1 lira
}

// Strategy: the "interface" is a function type, and any func of this shape is one.
type FeeRule func(Order) int

// Fee lets a FeeRule satisfy a one-method interface, like http.HandlerFunc.
func (f FeeRule) Fee(o Order) int { return f(o) }

func flat(kurus int) FeeRule { return func(Order) int { return kurus } }

// Decorators take a FeeRule and return one, so they stack with no base class.
func freeAbove(limit int, next FeeRule) FeeRule {
	return func(o Order) int {
		if o.Kurus >= limit {
			return 0
		}
		return next(o)
	}
}

func capped(most int, next FeeRule) FeeRule {
	return func(o Order) int { return min(next(o), most) }
}

// Adapter: a carrier SDK with its own shape (lira as float64, desi in, quote out).
type cargoSDK struct{ base, perDesi float64 }

func (c cargoSDK) Quote(desi float64) float64 { return c.base + c.perDesi*desi }

func fromSDK(c cargoSDK) FeeRule {
	return func(o Order) int { return int(math.Round(c.Quote(o.Desi) * 100)) }
}

// The consumer declares the port it needs; it has never heard of FeeRule.
type Pricer interface{ Fee(Order) int }

func lira(k int) string { return fmt.Sprintf("%d,%02d TL", k/100, k%100) }

func main() {
	carrier := fromSDK(cargoSDK{base: 39.90, perDesi: 12.50})
	rules := []struct {
		name string
		p    Pricer
	}{
		{"flat", flat(4990)},
		{"carrier", carrier},
		{"carrier, capped", capped(9990, carrier)},
		{"free over 500 TL", freeAbove(50000, capped(9990, carrier))},
	}
	for _, o := range []Order{{Desi: 2, Kurus: 45000}, {Desi: 6, Kurus: 72000}} {
		fmt.Printf("%.0f desi, %s\n", o.Desi, lira(o.Kurus))
		for _, r := range rules {
			fmt.Printf("  %-17s %10s\n", r.name, lira(r.p.Fee(o)))
		}
	}
}

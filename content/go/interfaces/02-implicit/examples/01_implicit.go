package main

import "fmt"

// Teller is declared at the consumer: it only asks for what it uses.
type Teller interface {
	Balance() float64
}

type Account struct{ bal float64 }

// Value receiver: both Account and *Account satisfy Teller.
func (a Account) Balance() float64 { return a.bal }

type Celsius float64

// Value receiver — Celsius satisfies fmt.Stringer implicitly.
func (c Celsius) String() string { return fmt.Sprintf("%.1f°C", c) }

type Gauge struct{ val int }

// Pointer receiver: only *Gauge (not Gauge) has Meter in its method set.
func (g *Gauge) Meter() int { return g.val }

type Meter interface{ Meter() int }

func main() {
	var t Teller = Account{21.5} // no implements keyword anywhere
	fmt.Println(Celsius(21.5))   // Stringer satisfied implicitly

	g := Gauge{72}
	var m Meter = &g // *Gauge satisfies Meter; Gauge alone would not compile
	fmt.Println("gauge at", m.Meter())

	// Both values satisfy Teller through their method sets:
	var t2 Teller = &Account{9.75}
	fmt.Println(t.Balance(), t2.Balance())
}

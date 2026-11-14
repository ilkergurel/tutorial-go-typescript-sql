package main

import (
	"fmt"
	"maps"
)

// PriceList is immutable by API: the map is unexported, nothing writes it
// after construction, and With returns a new list (copy-on-write).
type PriceList struct {
	prices map[string]int64 // kuruş
}

func NewPriceList(prices map[string]int64) PriceList {
	return PriceList{prices: maps.Clone(prices)} // keep no alias to the caller's map
}

func (p PriceList) Price(sku string) int64 { return p.prices[sku] }

func (p PriceList) With(sku string, kurus int64) PriceList {
	next := maps.Clone(p.prices)
	next[sku] = kurus
	return PriceList{prices: next}
}

// The trap, for comparison: a value receiver copies the struct, not the map.
func (p PriceList) SetInPlace(sku string, kurus int64) { p.prices[sku] = kurus }

type Order struct {
	ID     string
	Prices PriceList // the list the order was priced with
}

func main() {
	october := NewPriceList(map[string]int64{"ayran": 1000, "simit": 1250})
	order := Order{ID: "S-1", Prices: october}

	november := october.With("simit", 1500)
	fmt.Println("order S-1 simit:", order.Prices.Price("simit"))
	fmt.Println("current simit:  ", november.Price("simit"))

	copied := november // a new struct around the same map
	copied.SetInPlace("ayran", 0)
	fmt.Println("november ayran after writing to a copy:", november.Price("ayran"))
	fmt.Println("october ayran, isolated by With:       ", october.Price("ayran"))
}

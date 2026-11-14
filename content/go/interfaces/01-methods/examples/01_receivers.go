package main

import "fmt"

type Counter struct{ n int }

// Wrong on purpose: a value receiver gets a copy of the caller's Counter.
func (c Counter) IncCopy() { c.n++ }

// A pointer receiver points at the caller's Counter.
func (c *Counter) Inc() { c.n++ }

func main() {
	var c Counter
	c.IncCopy()
	fmt.Println("after IncCopy:", c.n)
	c.Inc() // c is addressable, so Go calls (&c).Inc()
	fmt.Println("after Inc:", c.n)

	counters := []Counter{{}, {}}
	for _, k := range counters {
		k.Inc() // k is a copy of the element
	}
	fmt.Println("range by value:", counters)
	for i := range counters {
		counters[i].Inc() // a slice element is addressable
	}
	fmt.Println("range by index:", counters)

	visits := map[string]Counter{"home": {}}
	// visits["home"].Inc() does not compile: map elements are not addressable.
	k := visits["home"]
	k.Inc()
	visits["home"] = k
	fmt.Println("map:", visits)
}

package main

import (
	"fmt"
	"iter"
)

type city struct {
	ID   int
	Name string
}

// page is one API response: the items plus an opaque cursor for the next
// page, empty on the last one.
type page struct {
	Items []city
	Next  string
}

// cities yields every city, calling fetch for the next page only when the
// loop asks for more. A failed fetch is yielded once, then the sequence ends.
func cities(fetch func(cursor string) (page, error)) iter.Seq2[city, error] {
	return func(yield func(city, error) bool) {
		cursor := ""
		for {
			p, err := fetch(cursor)
			if err != nil {
				yield(city{}, err)
				return
			}
			for _, c := range p.Items {
				if !yield(c, nil) {
					return // the caller broke out of its loop
				}
			}
			if p.Next == "" {
				return
			}
			cursor = p.Next
		}
	}
}

func main() {
	// A fake API: pages of three, ordered by ID. Each cursor is the last ID
	// of its page, base64-encoded: "Nw" is 7, "MzQ" is 34.
	api := map[string]page{
		"":    {[]city{{1, "Adana"}, {6, "Ankara"}, {7, "Antalya"}}, "Nw"},
		"Nw":  {[]city{{16, "Bursa"}, {21, "Diyarbakır"}, {34, "İstanbul"}}, "MzQ"},
		"MzQ": {[]city{{35, "İzmir"}, {42, "Konya"}}, ""},
	}
	requests := 0
	fetch := func(cursor string) (page, error) {
		requests++ // in real code: lesson 1's GET, JSON decoding, lesson 2's retries
		return api[cursor], nil
	}

	for c, err := range cities(fetch) {
		if err != nil {
			fmt.Println("error:", err)
			break
		}
		fmt.Println(c.ID, c.Name)
		if c.Name == "Bursa" {
			break // found it: the third page is never fetched
		}
	}
	fmt.Println("requests:", requests)
}

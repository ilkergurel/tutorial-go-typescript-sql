package main

import (
	"fmt"
	"math"
	"slices"
)

// Two ways to page a newest-first feed of post IDs, three per page.

// offsetPage is ORDER BY id DESC LIMIT 3 OFFSET $1: "skip n rows".
func offsetPage(feed []int, offset int) []int {
	return slices.Clone(feed[min(offset, len(feed)):min(offset+3, len(feed))])
}

// keysetPage is WHERE id < $1 ORDER BY id DESC LIMIT 3: "continue after
// the last row I saw". The cursor names a row, not a position.
func keysetPage(feed []int, before int) []int {
	var page []int
	for _, id := range feed {
		if id < before && len(page) < 3 {
			page = append(page, id)
		}
	}
	return page
}

func main() {
	for _, c := range []struct {
		name   string
		change func([]int) []int
	}{
		{"post 11 is published", func(f []int) []int { return slices.Insert(f, 0, 11) }},
		{"post 9 is deleted", func(f []int) []int { return slices.DeleteFunc(f, func(id int) bool { return id == 9 }) }},
	} {
		feed := []int{10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
		off1, key1 := offsetPage(feed, 0), keysetPage(feed, math.MaxInt)

		feed = c.change(feed) // someone else writes between our two requests

		off2, key2 := offsetPage(feed, len(off1)), keysetPage(feed, key1[len(key1)-1])
		fmt.Println(c.name, "between page 1 and page 2:")
		fmt.Println("  offset:", off1, off2)
		fmt.Println("  keyset:", key1, key2)
	}
}

package main

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

func main() {
	defaults := map[string]string{"lang": "tr", "theme": "light", "tz": "Europe/Istanbul"}
	user := map[string]string{"theme": "dark"}

	// Merge: start from a copy of the defaults, then overwrite with the user's.
	settings := maps.Clone(defaults)
	maps.Copy(settings, user)
	fmt.Println(settings)
	fmt.Println("defaults untouched:", defaults["theme"])

	// Keys and Values return iterators (Go 1.23); collect them as needed.
	fmt.Println(slices.Sorted(maps.Keys(settings)))
	fmt.Println(slices.Sorted(maps.Values(settings)))

	// Remove entries matching a predicate without a hand-written loop.
	maps.DeleteFunc(settings, func(k, v string) bool {
		return strings.HasPrefix(v, "Europe/")
	})
	want := map[string]string{"lang": "tr", "theme": "dark"}
	fmt.Println(settings, maps.Equal(settings, want))

	// Build a map from any key/value iterator: here, index -> element.
	byIndex := maps.Collect(slices.All([]string{"sıfır", "bir", "iki"}))
	fmt.Println(byIndex)

	// clear deletes every entry; the map stays usable (unlike a nil map).
	clear(settings)
	settings["lang"] = "en"
	fmt.Println(settings, len(settings))
}

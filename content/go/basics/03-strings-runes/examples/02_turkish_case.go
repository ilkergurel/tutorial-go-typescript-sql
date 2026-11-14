package main

import (
	"fmt"
	"strings"
	"unicode"
)

// lowerTurkish applies the Turkish alphabet's case rules: I -> ı, İ -> i.
// strings.ToLower never consults a locale, so Turkish has to be asked for.
func lowerTurkish(s string) string {
	return strings.ToLowerSpecial(unicode.TurkishCase, s)
}

func main() {
	words := []string{"ISTANBUL", "DIL", "DİL"}
	for _, w := range words {
		fmt.Printf("%s naive=%q turkish=%q\n", w, strings.ToLower(w), lowerTurkish(w))
	}

	// Why it matters: naive lowercase merges DIL and DİL into the same word,
	// the Turkish casing keeps them distinct — a real search-box bug.
	fmt.Printf("naive merge:   %v\n", strings.ToLower("DIL") == strings.ToLower("DİL"))
	fmt.Printf("turkish split: %v\n", lowerTurkish("DIL") == lowerTurkish("DİL"))
}

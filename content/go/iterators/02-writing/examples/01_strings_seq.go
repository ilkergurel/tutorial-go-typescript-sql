package main

import (
	"fmt"
	"iter"
	"slices"
	"strings"
	"unicode"
)

const stock = `ürün;etiketler
çay;sıcak,içecek
simit;fırın,susamlı
ayran;soğuk,içecek
`

func count(seq iter.Seq[string]) int {
	n := 0
	for range seq { // the zero-variable form: just count
		n++
	}
	return n
}

func main() {
	for line := range strings.Lines(stock) {
		fmt.Printf("%q\n", line) // every line keeps its "\n"
	}

	for line := range strings.Lines(stock) {
		name, tags, _ := strings.Cut(strings.TrimSuffix(line, "\n"), ";")
		if name == "ürün" {
			continue // header
		}
		fmt.Print(strings.ToUpperSpecial(unicode.TurkishCase, name), ":")
		for tag := range strings.SplitSeq(tags, ",") { // no []string built
			fmt.Print(" ", tag)
		}
		fmt.Println()
	}

	words := slices.Collect(strings.FieldsSeq("  bir   iki\tüç \n"))
	fmt.Printf("%q\n", words)

	lines := strings.Lines(stock) // documented as single-use
	fmt.Println(count(lines), count(lines))
}

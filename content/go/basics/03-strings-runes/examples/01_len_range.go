package main

import (
	"fmt"
	"unicode/utf8"
)

func main() {
	t := "DİL" // D, İ (two bytes), L

	fmt.Printf("len: %d runes: %d\n", len(t), utf8.RuneCountInString(t))

	// range decodes UTF-8: index is the BYTE offset of each rune.
	for i, r := range t {
		fmt.Printf("byte %d: rune %c\n", i, r)
	}

	// Indexing gives a byte, not a character: t[1] is the first byte of İ.
	b := t[1]
	fmt.Printf("t[1] as byte: %q\n", b)
	r, _ := utf8.DecodeRuneInString(t[1:])
	fmt.Printf("t[1] decoded: %c\n", r)
}

package main

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"
)

func main() {
	stock := map[string]int{"elma": 3, "armut": 0}

	n, ok := stock["armut"] // comma-ok: present, value 0
	fmt.Println("armut:", n, ok)
	n, ok = stock["kiraz"] // missing: also 0, but ok is false
	fmt.Println("kiraz:", n, ok)

	// Keys compare byte by byte, so normalize them first, with the right
	// rules: ılık (lukewarm) and ilik (bone marrow) are different words.
	naive, turkish := map[string]int{}, map[string]int{}
	for _, w := range []string{"ılık", "ILIK", "ilik", "İLİK"} {
		naive[strings.ToLower(w)]++
		turkish[strings.ToLowerSpecial(unicode.TurkishCase, w)]++
	}
	fmt.Println("naive:  ", naive) // fmt prints maps with sorted keys
	fmt.Println("turkish:", turkish)

	// range visits keys in a randomized order; sort them when order matters.
	for _, k := range slices.Sorted(maps.Keys(turkish)) {
		fmt.Println(k, turkish[k])
	}

	var empty map[string]int // nil map: reads and len work, writes panic
	fmt.Println(empty["x"], len(empty))
	fmt.Println(write(empty))
}

// write stores into m. The demo recovers the panic a nil map causes and
// returns it as an error, so the program can print it and go on.
func write(m map[string]int) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered: %v", r)
		}
	}()
	m["x"] = 1
	return nil
}

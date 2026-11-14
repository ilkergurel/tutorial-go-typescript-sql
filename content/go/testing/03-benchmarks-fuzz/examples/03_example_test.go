package main

import (
	"fmt"
	"strings"
	"unicode"
)

var ascii = strings.NewReplacer(
	"ı", "i", "ğ", "g", "ü", "u", "ş", "s", "ö", "o", "ç", "c",
	"â", "a", "î", "i", "û", "u", // kâr, Kâğıthane, millî
)

// Slug turns a Turkish title into a URL path segment.
func Slug(title string) string {
	var b strings.Builder
	dash := false
	for _, r := range ascii.Replace(strings.ToLower(title)) {
		switch {
		case 'a' <= r && r <= 'z', '0' <= r && r <= '9':
			b.WriteRune(r)
			dash = false
		case r == '\'' || r == '’' || unicode.Is(unicode.Mn, r):
			// apostrophes and combining marks vanish: İzmir'de -> izmirde
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

func ExampleSlug() {
	fmt.Println(Slug("Şanlıurfa'da Kış Turizmi"))
	fmt.Println(Slug("  IĞDIR — Kâğıthane  "))
	fmt.Println(Slug("i\u0307stanbul")) // lowercased by JavaScript's toLowerCase
	// Output:
	// sanliurfada-kis-turizmi
	// igdir-kagithane
	// istanbul
}

func ExampleSlug_plates() {
	plates := map[string]int{"İzmir": 35, "Muğla": 48, "Çanakkale": 17}
	for city, plate := range plates { // map order varies from run to run
		fmt.Println(plate, Slug(city))
	}
	// Unordered output:
	// 35 izmir
	// 48 mugla
	// 17 canakkale
}

// An example without an output comment is compiled, shown in the docs,
// and never run.
func ExampleSlug_untested() {
	fmt.Println(Slug("Ölüdeniz"))
}

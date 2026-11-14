package main

import "fmt"

type Point struct{ X, Y int }

type Celsius float64

// String makes Celsius a fmt.Stringer. Convert before formatting, or
// Sprintf("%v", c) would call String again and recurse forever.
func (c Celsius) String() string { return fmt.Sprintf("%.1f°C", float64(c)) }

func main() {
	p := Point{3, 4}
	fmt.Printf("%v | %+v | %#v | %T\n", p, p, p, p)

	t := Celsius(21.5)
	fmt.Printf("%v | %s | %.2f\n", t, t, t)

	city := "İzmir"
	fmt.Printf("%q | %+q | % x | %U\n", city, city, city[:2], []rune(city)[0])
	fmt.Printf("[%8s] [%-8s] len=%d\n", city, city, len(city))

	fmt.Printf("[%5d] [%-5d] [%05d] [%x] [%8.3f]\n", 42, 42, 42, 255, 3.14159)
	fmt.Println(map[string]int{"b": 2, "c": 3, "a": 1})

	// Mistakes compile and run: fmt prints a %! marker instead of failing.
	fmt.Println(fmt.Sprintf("%d", "42"))
	fmt.Println(fmt.Sprintf("%s and %s", "one"))
	fmt.Println(fmt.Sprintf("%s", "one", "two"))
}

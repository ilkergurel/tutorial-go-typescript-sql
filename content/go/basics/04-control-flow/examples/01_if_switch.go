package main

import (
	"fmt"
	"strconv"
)

func classify(input string) string {
	n, err := strconv.Atoi(input)
	if err != nil {
		return "not a number"
	}
	switch { // tagless switch: an if/else-if chain, first true case wins
	case n < 0:
		return "negative"
	case n == 0:
		return "zero"
	case n%2 == 0:
		return "even"
	default:
		return "odd"
	}
}

func main() {
	for _, in := range []string{"x", "-5", "0", "4", "7"} {
		fmt.Printf("%-2s -> %s\n", in, classify(in))
	}

	limits := map[string]int{"free": 3, "pro": 100}
	if v, ok := limits["team"]; ok { // v and ok exist only in this if/else
		fmt.Println("team limit:", v)
	} else {
		fmt.Println("no team plan; free limit is", limits["free"])
	}

	switch day := "Sat"; day { // no break needed: cases never fall through
	case "Sat", "Sun": // one case, several values
		fmt.Println(day, "is weekend")
	default:
		fmt.Println(day, "is a weekday")
	}

	switch grade := 95; {
	case grade >= 90:
		fmt.Println("A")
		fallthrough // jumps into the next body WITHOUT testing its condition
	case grade < 0:
		fmt.Println("this runs too, although grade < 0 is false")
	}
}

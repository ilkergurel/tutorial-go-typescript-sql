package main

import (
	"errors"
	"fmt"
)

func lookup(key string) (string, error) {
	if key == "" {
		return "", errors.New("empty key")
	}
	return "value-of-" + key, nil
}

func main() {
	var result string
	var err error
	cached := false

	if !cached {
		result, err := lookup("a") // BUG: := declares a new result and err
		fmt.Println("inner:", result, err)
	}
	fmt.Printf("outer after bug: %q %v\n", result, err)

	if !cached {
		result, err = lookup("b") // fix: = assigns the outer variables
	}
	fmt.Printf("outer after fix: %q %v\n", result, err)
}

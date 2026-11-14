package main

import (
	"errors"
	"fmt"
	"strconv"
)

// parsePort adds context to every error in one place: a deferred closure can
// read and replace named results after a return statement has set them.
func parsePort(s string) (port int, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("parse port %q: %w", s, err)
		}
	}()
	port, err = strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if port < 1 || port > 65535 {
		return 0, errors.New("out of range")
	}
	return port, nil
}

// safeDiv turns a panic into an ordinary error. recover only has an effect
// when called directly by a deferred function.
func safeDiv(a, b int) (q int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered: %v", r)
		}
	}()
	return a / b, nil
}

func main() {
	for _, s := range []string{"8080", "http", "70000"} {
		fmt.Println(parsePort(s))
	}
	fmt.Println(safeDiv(7, 2))
	fmt.Println(safeDiv(1, 0))
}

package main

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf8"
)

func describe(v any) string {
	switch x := v.(type) { // cases are tried top to bottom
	case nil:
		return "nil interface"
	case int, int64:
		return fmt.Sprintf("integer %v of type %T", x, x) // two types: x stays any
	case string:
		return fmt.Sprintf("string: %d bytes, %d runes", len(x), utf8.RuneCountInString(x))
	case error:
		return "error: " + x.Error()
	case fmt.Stringer:
		return "Stringer: " + x.String()
	default:
		return fmt.Sprintf("something else: %T", x)
	}
}

func mustInt(v any) (n int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	return v.(int), nil // the one-result form panics on a mismatch
}

func main() {
	var v any = "İzmir"
	s, ok := v.(string)
	fmt.Println(s, ok)
	n, ok := v.(int) // comma-ok: no panic, n is int's zero value
	fmt.Println(n, ok)
	_, err := mustInt(v)
	fmt.Println(err)

	for _, x := range []any{nil, 42, int64(7), "ağaç", errors.New("boom"), 90 * time.Second, 3.5} {
		fmt.Println(describe(x))
	}
}

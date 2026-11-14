package main

import (
	"encoding/json"
	"fmt"
	"go/token"
	"reflect"
)

// Customer mixes exported (upper-case) and unexported (lower-case) fields.
type Customer struct {
	Name     string
	City     string
	password string
}

func main() {
	c := Customer{Name: "Ayşe", City: "İzmir", password: "s3cret"}

	// encoding/json lives in another package, so it only sees exported fields.
	b, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))

	// fmt reads unexported fields through reflection to print them.
	fmt.Printf("%+v\n", c)

	for _, f := range reflect.VisibleFields(reflect.TypeFor[Customer]()) {
		fmt.Printf("field %-8s exported=%t\n", f.Name, f.IsExported())
	}

	// The rule is about the first character's Unicode category (Lu), not ASCII.
	for _, name := range []string{"Total", "total", "İlçe", "ılçe", "_Total", "日本"} {
		fmt.Printf("%-6s exported=%t\n", name, token.IsExported(name))
	}
}

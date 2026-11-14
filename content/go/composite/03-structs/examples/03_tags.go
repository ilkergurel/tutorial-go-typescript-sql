package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

type Customer struct {
	ID       int       `json:"id" db:"customer_id"`
	Name     string    `json:"name" db:"full_name"`
	Email    string    `json:"email,omitempty"`
	Joined   time.Time `json:"joined,omitempty"` // omitempty never omits a struct
	Birthday time.Time `json:"birthday,omitzero"`
	Password string    `json:"-"`
}

func main() {
	c := Customer{ID: 1, Name: "Ayşe Yılmaz", Password: "s3cret"}
	out, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))

	// Decoding matches keys case-insensitively when there is no exact match.
	var back Customer
	if err := json.Unmarshal([]byte(`{"ID": 2, "NAME": "Can"}`), &back); err != nil {
		panic(err)
	}
	fmt.Println(back.ID, back.Name)

	// A tag is a string attached to a field; libraries read it with reflect.
	for _, f := range reflect.VisibleFields(reflect.TypeFor[Customer]()) {
		if col, ok := f.Tag.Lookup("db"); ok {
			fmt.Printf("%s -> column %s\n", f.Name, col)
		}
	}
}

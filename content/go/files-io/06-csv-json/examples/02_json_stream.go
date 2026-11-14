package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// envelope decodes the common part now and keeps data as raw bytes until the
// type says what it is.
type envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type orderPlaced struct {
	ID         int   `json:"id"`
	TotalKurus int64 `json:"total_kurus"` // money as integer kuruş, never float64
}

type userRenamed struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

const events = `[
  {"type": "order.placed", "data": {"id": 7, "total_kurus": 125050}},
  {"type": "user.renamed", "data": {"id": 3, "name": "Işıl"}},
  {"type": "user.deleted", "data": {"id": 4}}
]`

func main() {
	dec := json.NewDecoder(strings.NewReader(events)) // any io.Reader: a file, an HTTP body

	if _, err := dec.Token(); err != nil { // consume the opening [
		log.Fatal(err)
	}
	for dec.More() { // one element at a time: memory stays flat for a 2 GB array
		var e envelope
		if err := dec.Decode(&e); err != nil {
			log.Fatal(err)
		}
		var err error
		switch e.Type {
		case "order.placed":
			var o orderPlaced
			err = json.Unmarshal(e.Data, &o)
			fmt.Printf("order %d: %d,%02d TL\n", o.ID, o.TotalKurus/100, o.TotalKurus%100)
		case "user.renamed":
			var u userRenamed
			err = json.Unmarshal(e.Data, &u)
			fmt.Printf("user %d is now %s\n", u.ID, u.Name)
		default:
			fmt.Printf("skipped %s, raw data kept: %s\n", e.Type, e.Data)
		}
		if err != nil {
			log.Fatal(err)
		}
	}
	if _, err := dec.Token(); err != nil { // the closing ]: also proves the array was complete
		log.Fatal(err)
	}
}

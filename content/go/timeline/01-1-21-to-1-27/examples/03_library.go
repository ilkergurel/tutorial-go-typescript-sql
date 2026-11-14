package main

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"sync"
	"uuid"
)

type Config struct {
	Name  string `json:"name"`
	Debug *bool  `json:"debug,omitempty"`
}

func main() {
	// Go 1.25: wg.Go replaces wg.Add(1); go func() { defer wg.Done(); ... }().
	squares := make([]int, 5)
	var wg sync.WaitGroup
	for i := range squares {
		wg.Go(func() { squares[i] = i * i })
	}
	wg.Wait()
	fmt.Println("wg.Go:        ", squares)

	// Go 1.26: new accepts an expression; no ptr helper or temp variable.
	cfg := Config{Name: "api", Debug: new(true)}
	fmt.Println("new(true):    ", *cfg.Debug)

	// Go 1.24: os.Root confines names to a directory. Go 1.26:
	// errors.AsType replaces declaring a target and calling errors.As.
	if err := os.Mkdir("public", 0o755); err != nil {
		log.Fatal(err)
	}
	root, err := os.OpenRoot("public")
	if err != nil {
		log.Fatal(err)
	}
	defer root.Close()
	_, err = root.Open("../.env")
	if pe, ok := errors.AsType[*fs.PathError](err); ok {
		fmt.Println("os.Root:      ", pe.Path, "->", pe.Err)
	}

	// Go 1.27: encoding/json/v2 is strict by default; v1 keeps its rules.
	in := []byte(`{"name": "a", "name": "b"}`)
	var c1, c2 Config
	fmt.Println("json v1:      ", json.Unmarshal(in, &c1), c1.Name)
	fmt.Println("json v2:      ", jsonv2.Unmarshal(in, &c2))

	// Go 1.27: package uuid. Version 7 embeds a timestamp and sorts by time.
	a, b := uuid.NewV7(), uuid.NewV7()
	fmt.Println("uuid.NewV7:   ", len(a.String()), a.String()[14:15], a.Compare(b) < 0)
	fmt.Println("uuid.Parse:   ", uuid.MustParse("{F81D4FAE-7DEC-11D0-A765-00A0C91E6BF6}"))
}

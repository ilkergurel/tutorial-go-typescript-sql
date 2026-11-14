package main

import (
	"fmt"
	"strings"
)

// Tags is a slice type. Add changes the length, so it needs *Tags, and
// String follows: one receiver kind per type.
type Tags []string

func (t *Tags) Add(tag string) { *t = append(*t, tag) }
func (t *Tags) String() string { return strings.Join(*t, ",") }

// Counts is a map type. A map value already refers to shared storage, so a
// value receiver can change its contents without a pointer.
type Counts map[string]int

func (c Counts) Add(key string) { c[key]++ }

// Tree treats a nil *Tree as the empty tree.
type Tree struct {
	Left, Right *Tree
	Val         int
}

func (t *Tree) Sum() int {
	if t == nil {
		return 0
	}
	return t.Left.Sum() + t.Val + t.Right.Sum()
}

func main() {
	var tags Tags // a nil slice: append handles it
	tags.Add("go")
	tags.Add("sql")
	fmt.Println(tags.String(), len(tags))

	c := Counts{} // a nil Counts would panic on the first Add
	c.Add("GET")
	c.Add("GET")
	c.Add("POST")
	fmt.Println(c)

	var empty *Tree
	t := &Tree{Val: 2, Left: &Tree{Val: 1}, Right: &Tree{Val: 3}}
	fmt.Println(empty.Sum(), t.Sum())
}

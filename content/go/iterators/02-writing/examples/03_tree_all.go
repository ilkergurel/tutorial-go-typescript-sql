package main

import (
	"cmp"
	"fmt"
	"iter"
	"maps"
	"slices"
)

type Tree[K cmp.Ordered, V any] struct{ root *node[K, V] }

type node[K cmp.Ordered, V any] struct {
	key         K
	val         V
	left, right *node[K, V]
}

func (t *Tree[K, V]) Put(k K, v V) { t.root = t.root.put(k, v) }

func (n *node[K, V]) put(k K, v V) *node[K, V] {
	switch {
	case n == nil:
		return &node[K, V]{key: k, val: v}
	case k < n.key:
		n.left = n.left.put(k, v)
	case k > n.key:
		n.right = n.right.put(k, v)
	default:
		n.val = v
	}
	return n
}

// All yields entries in key order: an in-order walk with yield in the middle.
func (t *Tree[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) { t.root.walk(yield) }
}

// walk reports whether to keep going, so one false unwinds the recursion.
func (n *node[K, V]) walk(yield func(K, V) bool) bool {
	if n == nil {
		return true
	}
	return n.left.walk(yield) && yield(n.key, n.val) && n.right.walk(yield)
}

func main() {
	var plates Tree[int, string]
	for k, v := range map[int]string{34: "İstanbul", 6: "Ankara", 35: "İzmir", 1: "Adana", 76: "Iğdır"} {
		plates.Put(k, v) // random insertion order, same in-order walk
	}
	for k, v := range plates.All() {
		fmt.Println(k, v)
		if k >= 34 {
			break
		}
	}
	byPlate := maps.Collect(plates.All()) // a Seq2 feeds the standard helpers
	fmt.Println(len(byPlate), slices.Sorted(maps.Values(byPlate)))
}

package main

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

var ids = []string{
	"go.basics.defer", "go.basics.pointers", "go.composite.structs", "go.http.handlers",
	"go.postgres.pgx", "sql.joins.lateral", "sql.search.turkish", "ts.react.query",
}

var (
	sink  string
	sinks []string
)

func concat() string {
	s := ""
	for _, id := range ids {
		s += id + "\n" // a new string every iteration
	}
	return s
}

func build(grow bool) string {
	var b strings.Builder
	if grow {
		b.Grow(len(ids) * 24) // one allocation, if the estimate is big enough
	}
	for _, id := range ids {
		b.WriteString(id)
		b.WriteByte('\n')
	}
	return b.String()
}

func collect(prealloc bool) []string {
	var out []string
	if prealloc {
		out = make([]string, 0, len(ids))
	}
	for _, id := range ids {
		out = append(out, id)
	}
	return out
}

func main() {
	measure := func(name string, f func()) {
		fmt.Printf("%-26s %v\n", name, testing.AllocsPerRun(1000, f))
	}
	measure("s += id", func() { sink = concat() })
	measure("strings.Builder", func() { sink = build(false) })
	measure("Builder after Grow", func() { sink = build(true) })
	measure("strings.Join", func() { sink = strings.Join(ids, "\n") })

	measure("append, no capacity", func() { sinks = collect(false) })
	measure("append into make(0, n)", func() { sinks = collect(true) })

	measure("fmt.Sprintf", func() { sink = fmt.Sprintf("%s:%d", ids[0], 1234) })
	buf := make([]byte, 0, 64)
	measure("strconv.AppendInt, reused", func() {
		buf = strconv.AppendInt(append(append(buf[:0], ids[0]...), ':'), 1234, 10)
	})

	runs := map[string]int{"go.basics.defer": 3}
	key := []byte("go.basics.defer")
	measure("m[string(key)]", func() { _ = runs[string(key)] })
}

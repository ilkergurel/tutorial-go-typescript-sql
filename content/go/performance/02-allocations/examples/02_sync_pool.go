package main

import (
	"bytes"
	"fmt"
	"strconv"
	"sync"
	"testing"
)

var (
	sink string
	ids  = []string{
		"go.basics.defer", "go.basics.pointers", "go.composite.structs", "go.http.handlers",
		"go.postgres.pgx", "sql.joins.lateral", "sql.search.turkish", "ts.react.query",
	}
)

// render writes one line per lesson into b and returns the text.
func render(b *bytes.Buffer, runs map[string]int) string {
	for _, id := range ids {
		b.WriteString(id)
		b.WriteByte(' ')
		b.Write(strconv.AppendInt(b.AvailableBuffer(), int64(runs[id]), 10))
		b.WriteByte('\n')
	}
	return b.String()
}

var bufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

func renderPooled(runs map[string]int) string {
	b := bufPool.Get().(*bytes.Buffer)
	b.Reset() // pooled objects come back dirty
	defer func() {
		if b.Cap() <= 64<<10 { // keep huge buffers out of the pool, as fmt does
			bufPool.Put(b)
		}
	}()
	return render(b, runs)
}

// A pool of slices: Put([]byte) boxes the slice header into an interface.
var slicePool = sync.Pool{New: func() any { return make([]byte, 0, 512) }}
var ptrPool = sync.Pool{New: func() any { b := make([]byte, 0, 512); return &b }}

func main() {
	runs := map[string]int{"go.basics.defer": 1200, "sql.joins.lateral": 95, "ts.react.query": 7}
	measure := func(name string, f func()) {
		fmt.Printf("%-24s %v\n", name, testing.AllocsPerRun(1000, f))
	}
	measure("new buffer per call", func() { sink = render(new(bytes.Buffer), runs) })
	measure("buffer from a Pool", func() { sink = renderPooled(runs) })

	measure("Pool of []byte", func() {
		buf := slicePool.Get().([]byte)
		sink = string(append(buf[:0], "go.basics.defer"...))
		slicePool.Put(buf[:0]) // allocates: a slice is not pointer-shaped
	})
	measure("Pool of *[]byte", func() {
		bp := ptrPool.Get().(*[]byte)
		*bp = append((*bp)[:0], "go.basics.defer"...)
		sink = string(*bp)
		ptrPool.Put(bp)
	})
	fmt.Println(renderPooled(runs) == render(new(bytes.Buffer), runs))
}

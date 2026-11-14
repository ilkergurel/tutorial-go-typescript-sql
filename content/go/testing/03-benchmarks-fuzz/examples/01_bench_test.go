package main

import (
	"strings"
	"testing"
)

// joinPlus builds the line with +=: every step copies everything so far.
func joinPlus(fields []string) string {
	s := ""
	for i, f := range fields {
		if i > 0 {
			s += ";"
		}
		s += f
	}
	return s
}

// joinBuilder sizes one buffer up front and writes into it.
func joinBuilder(fields []string) string {
	n := len(fields)
	for _, f := range fields {
		n += len(f)
	}
	var b strings.Builder
	b.Grow(n)
	for i, f := range fields {
		if i > 0 {
			b.WriteByte(';')
		}
		b.WriteString(f)
	}
	return b.String()
}

var cities = strings.Fields("İstanbul Ankara İzmir Bursa Antalya Konya Adana Şanlıurfa")

// Benchmarks run only with go test -bench; plain go test skips them.
func BenchmarkJoinPlus(b *testing.B) {
	for b.Loop() { // Go 1.24; timer, iteration count and dead-code guard in one
		joinPlus(cities)
	}
}

func BenchmarkJoinBuilder(b *testing.B) {
	for b.Loop() {
		joinBuilder(cities)
	}
}

// Allocation counts are exact, so a plain test can pin them.
func TestJoinAllocs(t *testing.T) {
	plus := testing.AllocsPerRun(100, func() { joinPlus(cities) })
	builder := testing.AllocsPerRun(100, func() { joinBuilder(cities) })
	t.Logf("allocations per call: += %v, Builder %v", plus, builder)
	if builder != 1 {
		t.Errorf("joinBuilder allocates %v times per call, want 1", builder)
	}
}

// testing.Benchmark runs a benchmark function without -bench (for about a
// second). Timings vary from run to run; the allocation count doesn't.
func TestBenchmarkByHand(t *testing.T) {
	r := testing.Benchmark(BenchmarkJoinBuilder)
	t.Logf("iterations > 0: %v, allocs/op: %d", r.N > 0, r.AllocsPerOp())
}

package main

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
)

// A named type documents intent, but an untyped string constant is assignable
// to it: "pased" compiles wherever a Status is expected. The switch checks.
type Status string

const (
	Passed Status = "passed"
	Failed Status = "failed"
)

type Run struct {
	Req, Test string
	Status    Status
}

type Tally struct {
	Req            string
	Passed, Failed int
}

func report(runs []Run) ([]Tally, error) {
	byReq := map[string]Tally{}
	for _, r := range runs {
		t := byReq[r.Req] // a missing key gives the zero Tally, ready to count
		t.Req = r.Req
		switch r.Status {
		case Passed:
			t.Passed++
		case Failed:
			t.Failed++
		default:
			return nil, fmt.Errorf("run %s: unknown status %q", r.Test, r.Status)
		}
		byReq[r.Req] = t // t is a copy: store it back
	}
	return slices.SortedFunc(maps.Values(byReq), func(a, b Tally) int {
		return cmp.Or(cmp.Compare(b.Failed, a.Failed), cmp.Compare(a.Req, b.Req))
	}), nil
}

func main() {
	tallies, err := report([]Run{
		{"REQ-LOGIN", "TC-1", Passed}, {"REQ-LOGIN", "TC-2", Failed},
		{"REQ-LOGIN", "TC-3", Passed}, {"REQ-EXPORT", "TC-4", Passed},
		{"REQ-SEARCH", "TC-5", Failed}, {"REQ-SEARCH", "TC-6", Failed},
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for _, t := range tallies {
		fmt.Printf("%-10s passed=%d failed=%d\n", t.Req, t.Passed, t.Failed)
	}

	_, err = report([]Run{{"REQ-LOGIN", "TC-7", "pased"}})
	fmt.Println("error:", err)
}

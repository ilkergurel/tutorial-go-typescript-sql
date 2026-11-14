package main

import (
	"cmp"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"iter"
	"os"
	"time"
)

// The wire types of TraceabilityReport in openapi.yaml.
type ReportTest struct {
	Code       string     `json:"code"`
	Result     *string    `json:"result"` // null: never run
	FinishedAt *time.Time `json:"finishedAt"`
}
type ReportItem struct {
	Code        string       `json:"code"`
	Title       string       `json:"title"`
	Depth       int          `json:"depth"`
	Coverage    string       `json:"coverage"`
	Tests       []ReportTest `json:"tests"`
	hasChildren bool         // unexported: not on the wire
}
type Summary struct {
	Requirements int `json:"requirements"`
	Counted      int `json:"counted"`
	Failing      int `json:"failing"`
	Uncovered    int `json:"uncovered"`
	Incomplete   int `json:"incomplete"`
	Passing      int `json:"passing"`
}

func writeReport(w io.Writer, project string, items iter.Seq2[ReportItem, error]) error {
	enc := jsontext.NewEncoder(w)
	if err := cmp.Or(enc.WriteToken(jsontext.BeginObject),
		enc.WriteToken(jsontext.String("project")), json.MarshalEncode(enc, map[string]string{"name": project}),
		enc.WriteToken(jsontext.String("requirements")), enc.WriteToken(jsontext.BeginArray)); err != nil {
		return err
	}
	var sum Summary
	tally := map[string]int{}
	for item, err := range items {
		if err != nil {
			return err // no closing brackets: a cut-off report must fail to parse
		}
		if err := json.MarshalEncode(enc, item); err != nil {
			return err
		}
		sum.Requirements++
		if !item.hasChildren || len(item.Tests) > 0 { // step 6: headings without tests don't count
			sum.Counted++
			tally[item.Coverage]++
		}
	}
	sum.Failing, sum.Uncovered, sum.Incomplete, sum.Passing = tally["failing"], tally["uncovered"], tally["incomplete"], tally["passing"]
	return cmp.Or(enc.WriteToken(jsontext.EndArray),
		enc.WriteToken(jsontext.String("summary")), json.MarshalEncode(enc, sum),
		enc.WriteToken(jsontext.EndObject))
}

func main() {
	passed, at := "passed", time.Date(2026, 11, 13, 6, 0, 0, 0, time.UTC)
	rows := func(yield func(ReportItem, error) bool) { // example 3's first four requirements
		for _, item := range []ReportItem{
			{"AUTH", "Kimlik doğrulama", 0, "uncovered", nil, true},
			{"AUTH-LOGIN", "Parola ile giriş", 1, "uncovered", nil, true},
			{"AUTH-LOGIN-2FA", "İki adımlı doğrulama", 2, "passing", []ReportTest{{"TC-1", &passed, &at}, {"TC-2", &passed, &at}}, false},
			{"AUTH-LOGIN-LOCKOUT", "Hesap kilitleme", 2, "incomplete", []ReportTest{{"TC-1", &passed, &at}, {"TC-4", nil, nil}}, false},
		} {
			if !yield(item, nil) {
				return
			}
		}
	}
	fmt.Println("error:", writeReport(os.Stdout, "Ödeme", rows))
}

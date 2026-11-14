package main

import (
	"errors"
	"maps"
	"strings"
	"testing"
)

// LEARN(go.testing.table-driven): table-driven tests, no assertion library.
func TestCountLevels(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]int
	}{
		{name: "empty input", input: "", want: map[string]int{}},
		{
			name: "mixed levels",
			input: "2026-11-01T09:00:00Z INFO server started\n" +
				"2026-11-01T09:00:05Z WARN slow query\n" +
				"2026-11-01T09:01:10Z ERROR db lost\n" +
				"2026-11-01T09:01:12Z INFO reconnected\n",
			want: map[string]int{"INFO": 2, "WARN": 1, "ERROR": 1},
		},
		{
			name:  "malformed lines skipped",
			input: "garbage\n\n2026-11-01T09:00:00Z INFO ok\n",
			want:  map[string]int{"INFO": 1},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := countLevels(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("countLevels: %v", err)
			}
			if !maps.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCountLevelsError(t *testing.T) {
	// A reader that always fails must surface an error, not silently return.
	_, err := countLevels(errReader{})
	if err == nil {
		t.Fatal("want error from failing reader, got nil")
	}
	if !errors.Is(err, errTest) {
		t.Errorf("want wrapped errTest, got %v", err)
	}
}

var errTest = errors.New("boom")

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errTest }

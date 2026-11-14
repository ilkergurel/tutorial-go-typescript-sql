package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// RateSource is declared by its consumer, the handler, with the one method
// it calls. Rate returns the price of one unit of cur in kuruş.
type RateSource interface {
	Rate(ctx context.Context, cur string) (int64, error)
}

func PriceHandler(rates RateSource) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /price/{cur}", func(w http.ResponseWriter, r *http.Request) {
		amount, err := strconv.ParseInt(r.URL.Query().Get("amount"), 10, 64)
		if err != nil || amount <= 0 {
			http.Error(w, "bad amount", http.StatusBadRequest)
			return
		}
		rate, err := rates.Rate(r.Context(), r.PathValue("cur"))
		if err != nil {
			http.Error(w, "rates unavailable", http.StatusBadGateway)
			return
		}
		fmt.Fprintf(w, "%d kuruş\n", amount*rate)
	})
	return mux
}

// fakeRates: canned answers, an injectable error, and a record of calls.
type fakeRates struct {
	rates map[string]int64
	err   error
	calls []string
}

func (f *fakeRates) Rate(_ context.Context, cur string) (int64, error) {
	f.calls = append(f.calls, cur)
	if f.err != nil {
		return 0, f.err
	}
	return f.rates[cur], nil
}

func TestPriceHandler(t *testing.T) {
	tests := []struct {
		name, target string
		srcErr       error
		code, calls  int
		body         string
	}{
		{"converts", "/price/USD?amount=3", nil, 200, 1, "12345 kuruş\n"},
		{"source down", "/price/USD?amount=3", errors.New("timeout"), 502, 1, "rates unavailable\n"},
		{"bad amount", "/price/USD?amount=-1", nil, 400, 0, "bad amount\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &fakeRates{rates: map[string]int64{"USD": 4115}, err: tt.srcErr}
			rec := httptest.NewRecorder()
			PriceHandler(src).ServeHTTP(rec, httptest.NewRequest("GET", tt.target, nil))
			if rec.Code != tt.code || rec.Body.String() != tt.body {
				t.Errorf("GET %s = %d %q, want %d %q", tt.target, rec.Code, rec.Body.String(), tt.code, tt.body)
			}
			if len(src.calls) != tt.calls {
				t.Errorf("Rate called %d times (%q), want %d", len(src.calls), src.calls, tt.calls)
			}
		})
	}
}

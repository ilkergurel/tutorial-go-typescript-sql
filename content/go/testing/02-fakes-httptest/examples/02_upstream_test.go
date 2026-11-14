package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

var ErrUnknownCurrency = errors.New("unknown currency")

// HTTPRates is the production RateSource, with the *http.Client injected.
type HTTPRates struct {
	Client  *http.Client
	BaseURL string
}

func (h HTTPRates) Rate(ctx context.Context, cur string) (int64, error) {
	u := h.BaseURL + "/v1/rates/" + url.PathEscape(cur)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	resp, err := h.Client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("rate %s: %w", cur, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return 0, ErrUnknownCurrency
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("rate %s: upstream %s", cur, resp.Status)
	}
	var body struct {
		Kurus int64 `json:"kurus"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, fmt.Errorf("rate %s: decode: %w", cur, err)
	}
	return body.Kurus, nil
}

func TestHTTPRates(t *testing.T) {
	// The fake upstream: a handler with canned answers, no network, no mock DSL.
	upstream := http.NewServeMux()
	upstream.HandleFunc("GET /v1/rates/{cur}", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("upstream got %s %s%s", r.Method, r.Host, r.URL.Path)
		switch r.PathValue("cur") {
		case "USD":
			fmt.Fprint(w, `{"kurus": 4115}`)
		case "EUR":
			http.Error(w, "bakımda", http.StatusServiceUnavailable)
		default:
			http.NotFound(w, r)
		}
	})
	srv := httptest.NewTestServer(t, upstream) // in-memory; closed by t.Cleanup

	// The production base URL, unchanged: srv.Client() routes any host to srv.
	rates := HTTPRates{Client: srv.Client(), BaseURL: "https://rates.example.com"}

	got, err := rates.Rate(t.Context(), "USD")
	if err != nil || got != 4115 {
		t.Errorf("Rate(USD) = %d, %v; want 4115, nil", got, err)
	}
	if _, err := rates.Rate(t.Context(), "XYZ"); !errors.Is(err, ErrUnknownCurrency) {
		t.Errorf("Rate(XYZ) error = %v, want ErrUnknownCurrency", err)
	}
	_, err = rates.Rate(t.Context(), "EUR")
	t.Logf("Rate(EUR) error: %v", err)
	if err == nil {
		t.Error("Rate(EUR) succeeded on a 503")
	}
}

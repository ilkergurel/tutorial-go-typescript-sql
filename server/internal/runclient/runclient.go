// Package runclient is the backend's HTTP client for the runner service.
//
// Docker mode only: there the backend never executes lesson code itself
// (ADR-0003); it validates, proxies to the sandboxed runner container, and
// records. Local mode runs lessons in-process via runexec instead (ADR-0008).
// Client satisfies httpapi.Runner, so handlers test against a fake.
package runclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"stacklab/server/internal/runapi"
)

// Client posts run requests to the runner service.
type Client struct {
	base *http.Client
	url  string
}

// New returns a client with a timeout above the runner's worst case
// (20s compile + 15s run) so a slow runner surfaces as an error, not a hang.
func New(baseURL string) *Client {
	return &Client{
		base: &http.Client{Timeout: 45 * time.Second},
		url:  baseURL,
	}
}

// Run forwards one validated request. The runner's refusals (busy, a runtime
// it can't run) come back as (*runapi.Error, nil) with the runner's status and
// detail, so the browser reads the same "server busy" as in local mode. A 500
// is the runner machinery failing, which runexec reports as an error; it
// becomes one here too, so both modes answer it with the same 502.
func (c *Client) Run(ctx context.Context, req runapi.Request) (*runapi.Result, *runapi.Error, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("runclient: encode: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/run", bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("runclient: request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.base.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("runclient: runner unreachable: %w", err)
	}
	defer resp.Body.Close()

	// The status picks the shape before anything is decoded. Decoding first
	// into Result is how the detail used to get lost: a problem body decodes
	// into Result without error (unknown fields are ignored) and nothing read
	// it again.
	if resp.StatusCode != http.StatusOK {
		refusal := runnerProblem(resp)
		if resp.StatusCode == http.StatusInternalServerError {
			return nil, nil, fmt.Errorf("runclient: runner failed: %s", refusal.Detail)
		}
		return nil, refusal, nil
	}
	var res runapi.Result
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, nil, fmt.Errorf("runclient: decode: %w", err)
	}
	return &res, nil, nil
}

// runnerProblem reads a non-200 reply. cmd/runner answers with a problem body
// that has runapi.Error's fields; anything else (a proxy's page, the runner
// mux's text/plain 404) keeps its status and, as detail, the start of its body.
func runnerProblem(resp *http.Response) *runapi.Error {
	refusal := &runapi.Error{Status: resp.StatusCode, Title: "runner error"}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
	var p runapi.Error
	if json.Unmarshal(body, &p) == nil {
		refusal.Detail = p.Detail
	} else {
		refusal.Detail = strings.TrimSpace(string(body))
	}
	if refusal.Detail == "" {
		refusal.Detail = http.StatusText(resp.StatusCode)
	}
	return refusal
}

// Health reports runner liveness for /readyz aggregation.
func (c *Client) Health(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := c.base.Do(httpReq)
	if err != nil {
		return fmt.Errorf("runner health: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("runner health: status %d", resp.StatusCode)
	}
	return nil
}

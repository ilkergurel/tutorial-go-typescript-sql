package runexec

import (
	"context"
	"net/http"
	"testing"

	"stacklab/server/internal/runapi"
)

// A runtime execute has no case for is a refusal the learner can read, not a
// machinery failure: the backend maps errors to 502 "runner or playground
// database unavailable", which would send them hunting for a broken database.
func TestUnsupportedRuntimeIsARefusal(t *testing.T) {
	res, refusal, err := New("", "").Run(context.Background(), runapi.Request{
		Runtime: runapi.Runtime("cobol"),
		Files:   map[string]string{"main.cob": "DISPLAY 'HI'."},
	})
	if err != nil {
		t.Fatalf("err = %v; want a refusal, not an error", err)
	}
	if res != nil {
		t.Fatalf("res = %+v; want nil", res)
	}
	if refusal == nil || refusal.Status != http.StatusNotImplemented || refusal.Detail == "" {
		t.Fatalf("refusal = %+v; want 501 with a detail", refusal)
	}
}

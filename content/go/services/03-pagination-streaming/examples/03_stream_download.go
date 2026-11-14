package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
)

// download streams url into name through a temp file and a hash, and
// renames only a complete, verified file into place.
func download(ctx context.Context, c *http.Client, url, name string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	tmp, err := os.CreateTemp(".", name+".*.part")
	if err != nil {
		return 0, err
	}
	defer os.Remove(tmp.Name()) // fails harmlessly after the rename
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), resp.Body)
	if err := errors.Join(err, tmp.Close()); err != nil {
		return n, err
	}
	got := "sha-256=:" + base64.StdEncoding.EncodeToString(h.Sum(nil)) + ":"
	if got != resp.Header.Get("Content-Digest") {
		return n, errors.New("digest mismatch")
	}
	return n, os.Rename(tmp.Name(), name)
}

func main() {
	body := strings.Repeat("İstanbul;Şanlıurfa;Diyarbakır\n", 300_000)
	sum := sha256.Sum256([]byte(body))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Digest", "sha-256=:"+base64.StdEncoding.EncodeToString(sum[:])+":")
		if r.URL.Path == "/cut.csv" {
			io.WriteString(w, body[:1<<20]) // a proxy cut the body short
			return
		}
		io.WriteString(w, body)
	}))
	defer srv.Close()

	for _, name := range []string{"cities.csv", "cut.csv"} {
		n, err := download(context.Background(), srv.Client(), srv.URL+"/"+name, name)
		fmt.Printf("%s: %d bytes, err=%v\n", name, n, err)
	}
	files, _ := filepath.Glob("*.csv*")
	fmt.Println("files left:", files)
}

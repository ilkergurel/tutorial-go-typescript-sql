package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
)

func session(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     "__Host-session", // browsers require Secure, Path=/ and no Domain
		Value:    value,            // an opaque ID; the session data stays on the server
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,                 // invisible to JavaScript, so XSS can't steal it
		Secure:   true,                 // only sent over HTTPS
		SameSite: http.SameSiteLaxMode, // not sent on cross-site POSTs or subrequests
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, session("s3cr3t-id", 3600))
		http.SetCookie(w, &http.Cookie{Name: "raw", Value: "Ayşe"}) // see stderr
		http.SetCookie(w, &http.Cookie{Name: "name", Value: base64.RawURLEncoding.EncodeToString([]byte("Ayşe"))})
	})
	mux.HandleFunc("GET /me", func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie("__Host-session"); err != nil {
			http.Error(w, "not logged in", http.StatusUnauthorized)
			return
		}
		raw, _ := r.Cookie("raw")
		c, _ := r.Cookie("name")
		name, _ := base64.RawURLEncoding.DecodeString(c.Value)
		fmt.Fprintf(w, "logged in: raw=%q decoded=%q\n", raw.Value, name)
	})
	mux.HandleFunc("POST /logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, session("", -1)) // MaxAge < 0 sends Max-Age=0: delete now
	})
	// Rejects unsafe cross-origin browser requests (CSRF), Go 1.25.
	srv := httptest.NewTLSServer(http.NewCrossOriginProtection().Handler(mux))
	defer srv.Close()

	client := srv.Client() // trusts the test server's certificate
	client.Jar, _ = cookiejar.New(nil)
	do := func(method, path, fetchSite string) {
		req, _ := http.NewRequest(method, srv.URL+path, nil)
		if fetchSite != "" {
			req.Header.Set("Sec-Fetch-Site", fetchSite) // what a browser would send
		}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(strings.TrimSpace(fmt.Sprintf("%s %s %d %s", method, path, resp.StatusCode, body)))
		for _, sc := range resp.Header.Values("Set-Cookie") {
			fmt.Println("    Set-Cookie:", sc)
		}
	}
	do("POST", "/login", "same-origin")
	do("GET", "/me", "")
	do("POST", "/logout", "cross-site") // a form on another site, posting here
	do("GET", "/me", "")
	do("POST", "/logout", "same-origin")
	do("GET", "/me", "")
}

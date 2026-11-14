// LEARN(fullstack.tracelite.step-01-architecture): one request, every layer it crosses, and where each name changes form.
package main

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Requirement struct { // the wire type, abridged: camelCase names, as openapi.yaml spells them
	ID        uuid.UUID  `json:"id"`
	ParentID  *uuid.UUID `json:"parentId"`
	CreatedAt time.Time  `json:"createdAt"`
}

func main() {
	db, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err == nil {
		_, err = db.Exec(context.Background(), `CREATE TABLE requirements (id uuid PRIMARY KEY, parent_id uuid, created_at timestamptz);
			INSERT INTO requirements VALUES ('01937b6e-0000-7000-8000-000000000001', NULL, '2026-11-13 09:30+03')`)
	}
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// The Go server (cmd/tracelite) owns /api: router, handler, pgx, PostgreSQL.
	api := http.NewServeMux()
	api.HandleFunc("GET /api/requirements/{requirementId}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("go       matched %q\n", r.Pattern)
		var req Requirement
		id, err := uuid.Parse(r.PathValue("requirementId"))
		if err == nil {
			err = db.QueryRow(r.Context(), "SELECT id, parent_id, created_at FROM requirements WHERE id = $1",
				id).Scan(&req.ID, &req.ParentID, &req.CreatedAt)
		}
		if err != nil { // malformed or unknown: the same route, a problem instead of a resource
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, `{"type":"about:blank","title":"Not Found","status":404}`)
			return
		}
		fmt.Printf("pgx      one row; created_at arrives in the zone %q\n", req.CreatedAt.Location())
		req.CreatedAt = req.CreatedAt.UTC()
		w.Header().Set("Content-Type", "application/json")
		json.MarshalWrite(w, req)
	})
	goServer := httptest.NewServer(api)
	defer goServer.Close()

	// The Vite dev server, reduced to its server.proxy rule (vite.config.ts).
	target, _ := url.Parse(goServer.URL)
	dev := http.NewServeMux()
	dev.Handle("/api/", httputil.NewSingleHostReverseProxy(target))
	vite := httptest.NewServer(dev)
	defer vite.Close()

	for _, path := range []string{"/api/requirements/01937b6e-0000-7000-8000-000000000001",
		"/api/requirements/01937b6e-0000-7000-8000-000000000999"} {
		fmt.Println("browser  GET", path)
		resp, err := http.Get(vite.URL + path)
		if err != nil {
			log.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("browser  %d %s %s\n\n", resp.StatusCode, resp.Header.Get("Content-Type"), body)
	}
}

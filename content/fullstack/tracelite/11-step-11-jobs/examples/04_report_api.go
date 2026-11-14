package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
)

func jobFile(db *pgxpool.Pool) http.HandlerFunc { // GET /api/jobs/{jobId}/file
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := uuid.Parse(r.PathValue("jobId")) // a malformed id finds no job
		var state, project string
		var file []byte
		var finished *time.Time
		err := db.QueryRow(r.Context(), `SELECT j.state, p.name, j.result, j.finished_at FROM jobs j
			JOIN projects p ON p.id = j.project_id WHERE j.id = $1 AND j.kind = 'report'`, id).Scan(&state, &project, &file, &finished)
		switch {
		case err != nil: // TraceLite tells "no such job" (404) apart from a failure (500)
			problem(w, http.StatusNotFound, "about:blank", "")
		case state != "done":
			problem(w, http.StatusConflict, "https://tracelite.example/problems/conflict", "the report is "+state+"; GET the job until it is done")
		default:
			name := project + " izlenebilirlik.json" // step 8's file name; its extension gives ServeContent the type
			w.Header().Set("Content-Disposition", "attachment; filename*=utf-8''"+url.PathEscape(name))
			http.ServeContent(w, r, name, *finished, bytes.NewReader(file))
		}
	}
}

func problem(w http.ResponseWriter, status int, typ, detail string) { // step 4's writeError, cut down
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"type":%q,"title":%q,"status":%d,"detail":%q}`, typ, http.StatusText(status), status, detail)
}

func main() {
	db, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err == nil {
		_, err = db.Exec(context.Background(), `CREATE TABLE projects (id uuid PRIMARY KEY, name text NOT NULL);
			CREATE TABLE jobs (id uuid PRIMARY KEY, project_id uuid REFERENCES projects, kind text, state text, result bytea, finished_at timestamptz);
			INSERT INTO projects VALUES ('01937b6e-0000-7000-8000-00000000000a', 'Ödeme');
			INSERT INTO jobs VALUES ('01937b6e-0000-7000-8000-0000000000c1', '01937b6e-0000-7000-8000-00000000000a', 'report', 'queued', NULL, NULL),
			  ('01937b6e-0000-7000-8000-0000000000c2', '01937b6e-0000-7000-8000-00000000000a', 'report', 'done',
			   '{"project":{"name":"Ödeme"},"requirements":[],"summary":{"requirements":0}}', now())`)
	}
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /api/jobs/{jobId}/file", jobFile(db))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	get(srv.URL+"/api/jobs/01937b6e-0000-7000-8000-0000000000c1/file", "")          // still queued
	get(srv.URL+"/api/jobs/01937b6e-0000-7000-8000-0000000000c2/file", "")          // done
	get(srv.URL+"/api/jobs/01937b6e-0000-7000-8000-0000000000c2/file", "bytes=29-") // a download manager resuming
}

func get(url, byteRange string) {
	req, _ := http.NewRequest("GET", url, nil)
	if byteRange != "" {
		req.Header.Set("Range", byteRange)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("%s, Content-Length %s, Content-Type %s, Content-Range %q\n  %s\n", resp.Status,
		resp.Header.Get("Content-Length"), resp.Header.Get("Content-Type"), resp.Header.Get("Content-Range"), body)
}

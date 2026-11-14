// LEARN(fullstack.tracelite.step-08-export): rows go from pgx to the client as they are read; once the status is sent, an error can only abort the response.
package main

import (
	"cmp"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func exportHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		project := r.PathValue("projectId") // a malformed id fails the lookup too: 404
		var name string
		err := db.QueryRow(r.Context(), "SELECT name FROM projects WHERE id = $1", project).Scan(&name)
		if err != nil {
			http.Error(w, "no such project", http.StatusNotFound) // writeError in TraceLite
			return
		}
		rows, _ := db.Query(r.Context(), `SELECT r.code, r.title, coalesce(p.code, ''), r.status, r.description FROM
			requirements r LEFT JOIN requirements p ON p.id = r.parent_id WHERE r.project_id = $1 ORDER BY r.code`, project)
		defer rows.Close()
		more := rows.Next() // a broken query fails here, while a 500 can still be sent
		if !more && rows.Err() != nil {
			http.Error(w, "export failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment",
			map[string]string{"filename": name + " gereksinimleri.csv"}))
		w.Write([]byte("\xEF\xBB\xBF")) // the BOM
		out := csv.NewWriter(w)
		out.Comma, out.UseCRLF = ';', true
		out.Write([]string{"code", "title", "parent", "status", "description"})
		for ; more && err == nil; more = rows.Next() {
			var code, title, parent, status, description string
			if err = rows.Scan(&code, &title, &parent, &status, &description); err == nil {
				err = out.Write([]string{code, title, parent, status, description})
			}
		}
		out.Flush()
		if err = cmp.Or(err, rows.Err(), out.Error()); err != nil {
			log.Printf("export of %s: %v", project, err)
			panic(http.ErrAbortHandler) // example 2: the download must fail, not end short
		}
	}
}

func main() {
	db, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	if _, err := db.Exec(context.Background(), setup); err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /api/projects/{projectId}/requirement-export", exportHandler(db))
	rec := httptest.NewRecorder() // the response a browser would save
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/projects/01937b6e-0000-7000-8000-00000000000a/requirement-export", nil))
	fmt.Println(rec.Code, rec.Header().Get("Content-Disposition"))
	for line := range strings.Lines(rec.Body.String()) {
		fmt.Printf("%q\n", line)
	}
}

const setup = `CREATE TABLE projects (id uuid PRIMARY KEY, name text NOT NULL); INSERT INTO projects VALUES ('01937b6e-0000-7000-8000-00000000000a', 'Ödeme');
CREATE TABLE requirements (id uuid PRIMARY KEY DEFAULT uuidv7(), project_id uuid NOT NULL, parent_id uuid, code text COLLATE "C" NOT NULL,
  title text NOT NULL, status text NOT NULL DEFAULT 'draft', description text NOT NULL DEFAULT '');
INSERT INTO requirements (project_id, code, title, status) SELECT id, 'AUTH', 'Kimlik doğrulama', 'approved' FROM projects;
INSERT INTO requirements (project_id, parent_id, code, title, description)
  SELECT project_id, id, 'AUTH-LOGIN', 'Parola ile giriş', 'Kullanıcı adı; parola' FROM requirements`

package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// exportHandler streams one line per row; abort decides what a failure after
// the first byte does.
func exportHandler(db *pgxpool.Pool, query string, abort bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, _ := db.Query(r.Context(), query)
		defer rows.Close() // runs while the abort panic unwinds too
		more := rows.Next()
		if !more && rows.Err() != nil {
			http.Error(w, "export failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		for ; more; more = rows.Next() {
			var code string
			if err := rows.Scan(&code); err != nil {
				break // rows.Err reports it
			}
			fmt.Fprintf(w, "%s\r\n", code)
		}
		if err := rows.Err(); err != nil {
			log.Printf("export: %v", err)
			if abort {
				panic(http.ErrAbortHandler)
			}
		}
	}
}

func main() {
	db, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	// Fails on its 1,500th row, after 1,499 have gone out, the way a statement
	// timeout or a restarted database breaks a real export.
	const failsLater = `SELECT 'REQ-' || (n + 0 / (1500 - n)) FROM generate_series(1, 2000) n`
	mux := http.NewServeMux()
	mux.Handle("/broken-query", exportHandler(db, "SELECT code FROM requirement", true)) // a typo
	mux.Handle("/returns", exportHandler(db, failsLater, false))
	mux.Handle("/aborts", exportHandler(db, failsLater, true))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, path := range []string{"/broken-query", "/returns", "/aborts"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			log.Fatal(err)
		}
		lines := bufio.NewScanner(resp.Body)
		n, last := 0, ""
		for ; lines.Scan(); n++ {
			last = lines.Text()
		}
		fmt.Printf("%-13s %s %v: %d lines, last %q, read error: %v\n",
			path, resp.Status, resp.TransferEncoding, n, last, lines.Err())
		resp.Body.Close()
	}
}

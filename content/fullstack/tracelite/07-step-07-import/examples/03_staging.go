package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Abridged: example 4 shows the whole table, with the id each row gets.
const staging = `CREATE TEMP TABLE import_rows (
	line integer, code text COLLATE "C", title text, parent text COLLATE "C"
) ON COMMIT DROP`

// stage streams a CSV body into import_rows with COPY. pgx asks for the next
// record whenever its send buffer has room, so memory holds about one record.
func stage(ctx context.Context, tx pgx.Tx, body io.Reader) (int64, error) {
	if _, err := tx.Exec(ctx, staging); err != nil {
		return 0, err
	}
	r := csv.NewReader(body) // UTF-8 already: example 2 decoded it
	r.Comma = ';'
	if _, err := r.Read(); err != nil { // the header, which example 2 checks
		return 0, err
	}
	var readErr error
	columns := []string{"line", "code", "title", "parent"}
	n, err := tx.CopyFrom(ctx, pgx.Identifier{"import_rows"}, columns,
		pgx.CopyFromFunc(func() ([]any, error) {
			rec, err := r.Read()
			if err == io.EOF {
				return nil, nil // no more rows: the COPY ends
			} else if err != nil {
				readErr = err
				return nil, err
			}
			line, _ := r.FieldPos(0)
			return []any{line, rec[0], rec[1], rec[2]}, nil
		}))
	if readErr != nil {
		return 0, fmt.Errorf("%w\n  pgx says: %v", readErr, err)
	}
	return n, err
}

func main() {
	ctx := context.Background()
	db, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close(ctx)
	const header = "code;title;parent\r\n"
	for _, body := range []string{
		header + "AUTH-LOGIN-2FA;İki adımlı doğrulama;AUTH-LOGIN\r\n" +
			"AUTH-LOGIN;Parola ile giriş;AUTH\r\n" +
			"EXPORT;\"Dışa aktarma; CSV\";\r\n",
		header + "REPORT;\"Rapor\r\nREPORT-PDF;PDF;REPORT\r\n", // a quote never closed
	} {
		tx, err := db.Begin(ctx) // the import's one transaction
		if err != nil {
			log.Fatal(err)
		}
		n, err := stage(ctx, tx, strings.NewReader(body))
		if err == nil {
			var titles string
			q := "SELECT string_agg(title, ' / ' ORDER BY line) FROM import_rows"
			err = tx.QueryRow(ctx, q).Scan(&titles)
			fmt.Printf("staged %d rows: %s\n", n, titles)
		}
		fmt.Println("error:", err)
		tx.Rollback(ctx) // here example 4's checks would run, then the INSERT
	}
}

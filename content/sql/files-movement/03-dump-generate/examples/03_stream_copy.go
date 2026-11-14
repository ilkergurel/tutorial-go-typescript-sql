package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	ctx := context.Background()
	// Two connections, as if to two databases: in real life dst gets another URL.
	src, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer src.Close(ctx)
	dst, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer dst.Close(ctx)

	_, err = src.Exec(ctx, `CREATE TABLE runs AS
		SELECT i AS id, 'TC-' || i AS case_code, (i % 7) * 1.25 AS seconds,
		       CASE WHEN i % 3 > 0 THEN 'ayşe' END AS tester
		FROM generate_series(1, 50000) AS i;
		CREATE TABLE runs_copy (LIKE runs)`)
	if err != nil {
		log.Fatal(err)
	}

	// The text format: tab-separated, \N for NULL. A plain pg_dump stores
	// each table's rows like this, between COPY ... FROM stdin; and \.
	var head bytes.Buffer
	_, err = src.PgConn().CopyTo(ctx, &head, "COPY (SELECT * FROM runs ORDER BY id LIMIT 3) TO STDOUT")
	fmt.Printf("%q %v\n", head.String(), err)

	// Stream one table into the other: no file, and io.Pipe holds no data.
	pr, pw := io.Pipe()
	done := make(chan error, 1)
	go func() {
		_, err := src.PgConn().CopyTo(ctx, pw, "COPY runs TO STDOUT")
		pw.CloseWithError(err) // nil: the reader sees EOF and the COPY completes
		done <- err
	}()
	tag, err := dst.PgConn().CopyFrom(ctx, pr, "COPY runs_copy FROM STDIN")
	pr.CloseWithError(err) // if CopyFrom stopped early, unblock the writer
	fmt.Println(tag, err, <-done)

	var same bool
	err = dst.QueryRow(ctx, `SELECT NOT EXISTS (TABLE runs EXCEPT ALL TABLE runs_copy)
		AND NOT EXISTS (TABLE runs_copy EXCEPT ALL TABLE runs)`).Scan(&same)
	fmt.Println("identical:", same, err)
}

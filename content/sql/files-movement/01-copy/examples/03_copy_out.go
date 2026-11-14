package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, `CREATE TABLE runs (case_code text, result text, note text, seconds numeric(6,2));
		INSERT INTO runs VALUES ('TC-1', 'geçti', NULL, 1.50),
		  ('TC-2', 'kaldı', E'tab\t"tırnak"', 12.25), ('TC-3', 'geçti', '', 0.75)`)
	if err != nil {
		log.Fatal(err)
	}

	// CopyTo streams the COPY data into any io.Writer: a file, an HTTP
	// response, a gzip.Writer. Here a buffer, printed with %q.
	export := func(label, prefix, sql string) {
		buf := bytes.NewBufferString(prefix)
		tag, err := conn.PgConn().CopyTo(ctx, buf, sql)
		fmt.Printf("%s: %v %v\n  %q\n", label, tag, err, buf.String())
	}
	export("text", "", "COPY (SELECT * FROM runs ORDER BY case_code) TO STDOUT")
	export("csv", "", "COPY (SELECT * FROM runs ORDER BY case_code) TO STDOUT WITH (FORMAT csv, HEADER)")
	export("excel", "\xEF\xBB\xBF", `COPY (SELECT case_code AS "Test", result AS "Sonuç",
		  replace(seconds::text, '.', ',') AS "Süre (sn)" FROM runs ORDER BY case_code)
		TO STDOUT WITH (FORMAT csv, HEADER, DELIMITER ';')`)
}

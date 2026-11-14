package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const file = "case_code;run_on;seconds\n" +
	"TC-1;2026-11-09;1.5\n" +
	"TC-2;09.11.2026;2\n" +
	"TC-3;13.11.2026;2\n" +
	"TC-4;2026-11-10;0,75\n" +
	"TC-5;2026-11-10;3\n"

func main() {
	ctx := context.Background()
	cfg, err := pgx.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	cfg.RuntimeParams["DateStyle"] = "ISO, MDY" // the built-in default; initdb may pick another
	cfg.OnNotice = func(_ *pgconn.PgConn, n *pgconn.Notice) { fmt.Println("  NOTICE:", n.Message) }
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, `CREATE TABLE runs (case_code text NOT NULL, run_on date NOT NULL,
		seconds numeric(6,2) NOT NULL CHECK (seconds >= 0))`)
	if err != nil {
		log.Fatal(err)
	}
	load := func(data, opts string) {
		fmt.Println(opts)
		sql := "COPY runs FROM STDIN WITH (FORMAT csv, HEADER, DELIMITER ';', " + opts + ")"
		tag, err := conn.PgConn().CopyFrom(ctx, strings.NewReader(data), sql)
		if err != nil {
			fmt.Printf("  %v\n", err)
			return
		}
		fmt.Printf("  %v\n", tag)
	}
	load(file, "ON_ERROR ignore, LOG_VERBOSITY verbose")
	load(file, "ON_ERROR ignore, REJECT_LIMIT 1")
	load(file+"TC-6;2026-11-11;-1\n", "ON_ERROR ignore")
	load(file+"TC-7;2026-11-11\n", "ON_ERROR ignore")

	rows, _ := conn.Query(ctx, "SELECT format('%s %s %s', case_code, run_on, seconds) FROM runs ORDER BY case_code")
	loaded, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("loaded:", strings.Join(loaded, ", "))
}

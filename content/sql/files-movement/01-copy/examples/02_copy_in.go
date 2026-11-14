package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

func main() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, "CREATE TABLE requirements (code text PRIMARY KEY, title text, priority int)")
	if err != nil {
		log.Fatal(err)
	}

	// What psql's \copy does: send COPY ... FROM STDIN, then stream the bytes.
	load := func(label, data, extra string) {
		sql := "COPY requirements FROM STDIN WITH (FORMAT csv, HEADER MATCH, DELIMITER ';'" + extra + ")"
		tag, err := conn.PgConn().CopyFrom(ctx, strings.NewReader(data), sql)
		if err != nil {
			fmt.Println(label, err)
			return
		}
		fmt.Println(label, tag)
	}
	load("utf-8:  ", "code;title;priority\n"+
		"REQ-1;Giriş ekranı;1\n"+
		"REQ-2;\"Rapor; PDF\";\n"+
		"REQ-3;\"\";2\n", "")
	load("bom:    ", "\xEF\xBB\xBFcode;title;priority\r\nREQ-4;Arama;1\r\n", "")
	legacy := "code;title;priority\r\nREQ-5;\xddhracat \xfeablonu;3\r\n" // windows-1254
	load("1254:   ", legacy, "")
	load("1254+enc", legacy, ", ENCODING 'WIN1254'")

	rows, _ := conn.Query(ctx, `SELECT format('%s | %s | %s', code, quote_nullable(title),
		coalesce(priority::text, 'NULL')) FROM requirements ORDER BY code`)
	lines, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(strings.Join(lines, "\n"))
}

// LEARN(design.architecture.security): the input parameters can't cover, and the error message that leaks.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

// titles binds the owner as a parameter. ORDER BY can't take a parameter, so
// the sort column arrives from the query string and is concatenated.
func titles(ctx context.Context, conn *pgx.Conn, owner, sort string) ([]string, error) {
	rows, _ := conn.Query(ctx, "SELECT title FROM notes WHERE owner = $1 ORDER BY "+sort, owner)
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

var ErrBadSort = errors.New("unknown sort key")

// sortSQL is the allow-list: public keys map to SQL the program wrote itself.
var sortSQL = map[string]string{"title": "title", "newest": "created_at DESC"}

func titlesSafe(ctx context.Context, conn *pgx.Conn, owner, key string) ([]string, error) {
	col, ok := sortSQL[key]
	if !ok {
		return nil, ErrBadSort // a 400 for the client, before any SQL exists
	}
	return titles(ctx, conn, owner, col)
}

func main() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, `
		CREATE TABLE notes (owner text, title text, created_at date);
		CREATE TABLE api_keys (owner text, secret text);
		INSERT INTO notes VALUES ('ayse', 'Defer', '2026-11-01'), ('ayse', 'Joins', '2026-11-12');
		INSERT INTO api_keys VALUES ('admin', 'sk_live_7Hq2')`)
	if err != nil {
		log.Fatal(err)
	}

	for _, sort := range []string{
		"title",
		"title; DROP TABLE api_keys", // stacked query
		"(SELECT CAST(secret AS int) FROM api_keys LIMIT 1)", // error-based
	} {
		got, err := titles(ctx, conn, "ayse", sort)
		fmt.Printf("concatenated ORDER BY %s\n  -> %v %v\n", sort, got, err)
	}
	_, err = titlesSafe(ctx, conn, "ayse", "(SELECT CAST(secret AS int) FROM api_keys LIMIT 1)")
	fmt.Println("allow-listed, unknown key\n  ->", err)
	got, err := titlesSafe(ctx, conn, "ayse", "newest")
	fmt.Println("allow-listed, newest\n  ->", got, err)
}

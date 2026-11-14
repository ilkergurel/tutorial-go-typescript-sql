package main

import (
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
	_, err = conn.Exec(ctx, `
		CREATE TABLE accounts (owner text PRIMARY KEY, balance int NOT NULL);
		INSERT INTO accounts VALUES ('Ayşe', 100), ('Çağrı', 20), ('Işıl', 75)`)
	if err != nil {
		log.Fatal(err)
	}

	owner := "nobody' OR 'a' = 'a" // straight from a form field

	// A parameter is a value: it can only ever be compared with owner.
	tag, err := conn.Exec(ctx, "UPDATE accounts SET balance = 0 WHERE owner = $1", owner)
	fmt.Println("parameter:   ", tag.RowsAffected(), "rows changed", err)

	// Concatenation makes the input part of the SQL text.
	tag, err = conn.Exec(ctx, "UPDATE accounts SET balance = 0 WHERE owner = '"+owner+"'")
	fmt.Println("concatenated:", tag.RowsAffected(), "rows changed", err)

	// Table and column names can't be parameters: quote them as identifiers.
	table := `accounts"; DROP TABLE accounts; --`
	query := "SELECT count(*) FROM " + pgx.Identifier{table}.Sanitize()
	fmt.Println(query)
	var n int
	fmt.Println(conn.QueryRow(ctx, query).Scan(&n))
	err = conn.QueryRow(ctx, "SELECT count(*) FROM accounts").Scan(&n)
	fmt.Println("accounts left:", n, err)
}

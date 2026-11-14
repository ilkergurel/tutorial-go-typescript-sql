package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const insert = "INSERT INTO users (email) VALUES ($1)"

func main() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, `
		CREATE TABLE users (email text PRIMARY KEY);
		INSERT INTO users VALUES ('ayse@example.com')`)
	if err != nil {
		log.Fatal(err)
	}
	emails := []string{"ayse@example.com", "isil@example.com"}

	// Wrong: log the error and carry on inside the same transaction.
	err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		for _, e := range emails {
			if _, err := tx.Exec(ctx, insert, e); err != nil {
				fmt.Println("  ignored:", err)
			}
		}
		return nil
	})
	fmt.Println("commit:", err, errors.Is(err, pgx.ErrTxCommitRollback))

	// Right: a savepoint around the statement that may fail.
	err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		for _, e := range emails {
			err := pgx.BeginFunc(ctx, tx, func(sp pgx.Tx) error { // SAVEPOINT
				_, err := sp.Exec(ctx, insert, e)
				return err
			})
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				fmt.Println("  skipped duplicate", e)
			} else if err != nil {
				return err
			}
		}
		return nil
	})
	fmt.Println("commit:", err)

	rows, _ := conn.Query(ctx, "SELECT email FROM users ORDER BY email")
	fmt.Println(pgx.CollectRows(rows, pgx.RowTo[string]))
}

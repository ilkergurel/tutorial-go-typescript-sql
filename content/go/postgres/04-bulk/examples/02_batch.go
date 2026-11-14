package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const insert = "INSERT INTO tags (name) VALUES ($1) RETURNING id"

func main() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, "CREATE TABLE tags (id int GENERATED ALWAYS AS IDENTITY, name text UNIQUE NOT NULL)")
	if err != nil {
		log.Fatal(err)
	}

	// Three statements, one round trip. Each callback runs as its result arrives.
	b := &pgx.Batch{}
	for _, name := range []string{"go", "pgx", "çay"} {
		b.Queue(insert, name).QueryRow(func(row pgx.Row) error {
			var id int
			err := row.Scan(&id)
			fmt.Println("inserted", name, "as", id)
			return err
		})
	}
	err = conn.SendBatch(ctx, b).Close() // Close waits for every result
	fmt.Println("batch 1:", err)

	// A batch runs in one implicit transaction: the failure undoes "sql" too.
	b = &pgx.Batch{}
	b.Queue(insert, "sql")
	b.Queue(insert, "go") // duplicate
	b.Queue("DELETE FROM tags").Exec(func(ct pgconn.CommandTag) error {
		fmt.Println("never runs:", ct)
		return nil
	})
	err = conn.SendBatch(ctx, b).Close()
	fmt.Println("batch 2:", err)

	var names []string
	err = conn.QueryRow(ctx, "SELECT array_agg(name ORDER BY id) FROM tags").Scan(&names)
	fmt.Println("tags:", names, err)
}

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib" // its init registers the "pgx" driver
)

func main() {
	ctx := context.Background()
	url := os.Getenv("DATABASE_URL")

	db, err := sql.Open("pgx", url) // *sql.DB is a pool too, not a connection
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	fmt.Println("database/sql MaxOpenConnections:", db.Stats().MaxOpenConnections, "(0 = unlimited)")
	db.SetMaxOpenConns(4)

	var tags []string
	err = db.QueryRowContext(ctx, "SELECT ARRAY['go', 'pgx', $1]", "çay").Scan(&tags)
	fmt.Println("database/sql:", tags, err)

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	err = pool.QueryRow(ctx, "SELECT ARRAY['go', 'pgx', $1]", "çay").Scan(&tags)
	fmt.Println("pgx:         ", tags, err)

	// COPY, batches and LISTEN are not in database/sql: reach the *pgx.Conn.
	conn, err := db.Conn(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	err = conn.Raw(func(driverConn any) error {
		c := driverConn.(*stdlib.Conn).Conn()
		if _, err := c.Exec(ctx, "CREATE TABLE cities (name text)"); err != nil {
			return err
		}
		n, err := c.CopyFrom(ctx, pgx.Identifier{"cities"}, []string{"name"},
			pgx.CopyFromRows([][]any{{"İzmir"}, {"Iğdır"}}))
		fmt.Println("copied through Raw:", n)
		return err
	})
	if err != nil {
		log.Fatal(err)
	}

	// A library that insists on *sql.DB (goose) can borrow the pgx pool.
	shared := stdlib.OpenDBFromPool(pool)
	defer shared.Close()
	var count int
	err = shared.QueryRowContext(ctx, "SELECT count(*) FROM cities").Scan(&count)
	fmt.Println("rows seen through the shared pool:", count, err)
}

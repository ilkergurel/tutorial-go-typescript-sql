package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Contact struct {
	Name  string      // NOT NULL column: a plain type is honest
	Phone *string     // nullable: nil means NULL
	Email pgtype.Text // nullable: Valid == false means NULL
}

func main() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `
		CREATE TABLE contacts (name text NOT NULL, phone text, email text);
		CREATE TABLE orders (customer text NOT NULL, qty int NOT NULL)`)
	if err != nil {
		log.Fatal(err)
	}
	var unknown *string // a nil pointer is sent as NULL
	_, err = conn.Exec(ctx, "INSERT INTO contacts VALUES ($1, $2, $3), ($4, $5, $6)",
		"Ömer", unknown, "omer@example.com", "Gül", "", nil)
	if err != nil {
		log.Fatal(err)
	}

	var phone string
	err = conn.QueryRow(ctx, "SELECT phone FROM contacts WHERE name = $1", "Ömer").Scan(&phone)
	fmt.Println("into string:", err)

	rows, _ := conn.Query(ctx, "SELECT name, phone, email FROM contacts ORDER BY name")
	contacts, err := pgx.CollectRows(rows, pgx.RowToStructByName[Contact])
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range contacts {
		p := "NULL"
		if c.Phone != nil {
			p = strconv.Quote(*c.Phone)
		}
		fmt.Printf("%-4s phone=%-4s email=%q valid=%t\n", c.Name, p, c.Email.String, c.Email.Valid)
	}

	// An aggregate over zero rows is NULL, not 0.
	var total int64
	err = conn.QueryRow(ctx, "SELECT sum(qty) FROM orders WHERE customer = $1", "Ömer").Scan(&total)
	fmt.Println("sum into int64:", err)
	err = conn.QueryRow(ctx, "SELECT coalesce(sum(qty), 0) FROM orders WHERE customer = $1", "Ömer").Scan(&total)
	fmt.Println("with coalesce:", total, err)
}

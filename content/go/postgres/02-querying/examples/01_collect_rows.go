package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

type User struct {
	ID        int64
	Name      string
	City      string
	BirthYear int32 // matches birth_year: case and underscores are ignored
}

func main() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `
		CREATE TABLE users (id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			name text NOT NULL, city text NOT NULL, birth_year int NOT NULL);
		INSERT INTO users (name, city, birth_year)
		VALUES ('Ayşe', 'İzmir', 1990), ('Çağrı', 'Ankara', 1985), ('Işıl', 'İzmir', 2001)`)
	if err != nil {
		log.Fatal(err)
	}

	const byCity = "SELECT id, name, city, birth_year FROM users WHERE city = $1 ORDER BY id"
	rows, _ := conn.Query(ctx, byCity, "İzmir") // errors resurface from CollectRows
	users, err := pgx.CollectRows(rows, pgx.RowToStructByName[User])
	fmt.Printf("%+v %v\n", users, err)

	rows, _ = conn.Query(ctx, "SELECT name FROM users ORDER BY id")
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	fmt.Println(names, err)

	rows, _ = conn.Query(ctx, "SELECT id, name, city, birth_year FROM users WHERE id = $1", 99)
	_, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[User])
	fmt.Println(err, errors.Is(err, pgx.ErrNoRows), errors.Is(err, sql.ErrNoRows))

	// Someone adds a column; SELECT * now returns a field User doesn't have.
	if _, err := conn.Exec(ctx, "ALTER TABLE users ADD COLUMN phone text"); err != nil {
		log.Fatal(err)
	}
	rows, _ = conn.Query(ctx, "SELECT * FROM users ORDER BY id")
	_, err = pgx.CollectRows(rows, pgx.RowToStructByName[User])
	fmt.Println(err)
}

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// db.go, hand-written here in the shape `sqlc generate` emits (sql_package: pgx/v5).
type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func New(db DBTX) *Queries { return &Queries{db: db} }

type Queries struct{ db DBTX }

func (q *Queries) WithTx(tx pgx.Tx) *Queries { return &Queries{db: tx} }

// models.go: one struct per table, nullable columns as pgtype values.
type User struct {
	ID    int64
	Email string
	City  string
	Phone pgtype.Text
}

// query.sql.go: the query from query.sql, name comment included.
const listUsersByCity = `-- name: ListUsersByCity :many
SELECT id, email, city, phone FROM users WHERE city = $1 ORDER BY email
`

func (q *Queries) ListUsersByCity(ctx context.Context, city string) ([]User, error) {
	rows, err := q.db.Query(ctx, listUsersByCity, city)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []User
	for rows.Next() {
		var i User
		if err := rows.Scan(&i.ID, &i.Email, &i.City, &i.Phone); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// Your code: no SQL, no Scan. (The CREATE TABLE stands in for the migrations.)
func main() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, `
		CREATE TABLE users (id bigint PRIMARY KEY, email text NOT NULL, city text NOT NULL, phone text);
		INSERT INTO users VALUES (1, 'ilker@example.com', 'İzmir', NULL),
			(2, 'ayse@example.com', 'İzmir', '+90 232 000 00 00')`); err != nil {
		log.Fatal(err)
	}

	q := New(conn) // a *pgx.Conn, a *pgxpool.Pool or a pgx.Tx: all satisfy DBTX
	for _, city := range []string{"İzmir", "Muğla"} {
		users, err := q.ListUsersByCity(ctx, city)
		fmt.Printf("%s: %v nil=%t err=%v\n", city, users, users == nil, err)
	}
}

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrEmailTaken = errors.New("email already registered")

// userCreator is declared by its consumer, signup, with the one method it uses.
type userCreator interface {
	Create(ctx context.Context, email string) (int64, error)
}

func signup(ctx context.Context, users userCreator, emails ...string) {
	for _, e := range emails {
		id, err := users.Create(ctx, e)
		fmt.Printf("  %-16s id=%d err=%v\n", e, id, err)
	}
}

// pgUsers is the repository: SQL and driver errors stay inside it.
type pgUsers struct{ pool *pgxpool.Pool }

func (r pgUsers) Create(ctx context.Context, email string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, "INSERT INTO users (email) VALUES ($1) RETURNING id", email).Scan(&id)
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
		return 0, ErrEmailTaken // unique_violation, translated at the boundary
	}
	return id, err
}

// memUsers is a fake: a map that believes it knows the database's rules.
type memUsers map[string]int64

func (m memUsers) Create(_ context.Context, email string) (int64, error) {
	if _, ok := m[email]; ok {
		return 0, ErrEmailTaken
	}
	m[email] = int64(len(m) + 1)
	return m[email], nil
}

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	_, err = pool.Exec(ctx, `
		CREATE TABLE users (id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, email text NOT NULL);
		CREATE UNIQUE INDEX users_email_key ON users (lower(email))`)
	if err != nil {
		log.Fatal(err)
	}

	emails := []string{"ayse@example.com", "Ayse@Example.com"}
	fmt.Println("fake:")
	signup(ctx, memUsers{}, emails...)
	fmt.Println("postgres:")
	signup(ctx, pgUsers{pool: pool}, emails...)
}

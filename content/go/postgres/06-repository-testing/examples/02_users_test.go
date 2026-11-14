package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrEmailTaken = errors.New("email already registered")

// querier is satisfied by *pgxpool.Pool in production and by pgx.Tx in tests.
type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type Users struct{ db querier }

func (r Users) Create(ctx context.Context, email string) error {
	_, err := r.db.Exec(ctx, "INSERT INTO users (email) VALUES ($1)", email)
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
		return fmt.Errorf("create %s: %w", email, ErrEmailTaken)
	}
	return err
}

var pool *pgxpool.Pool // set by TestMain
func TestMain(m *testing.M) {
	if url := os.Getenv("DATABASE_URL"); url != "" { // unset: tests skip themselves
		var err error
		pool, err = pgxpool.New(context.Background(), url)
		if err == nil { // stands in for running the migrations once
			_, err = pool.Exec(context.Background(), `CREATE TABLE users (email text NOT NULL);
				CREATE UNIQUE INDEX users_email_key ON users (lower(email))`)
		}
		if err != nil {
			log.Fatal(err)
		}
		defer pool.Close()
	}
	m.Run()
}

// txUsers gives a test its own transaction, rolled back when the test ends.
func txUsers(t *testing.T) Users {
	if pool == nil {
		t.Skip("DATABASE_URL not set")
	}
	tx, err := pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tx.Rollback(context.Background()) }) // t.Context() is canceled by now
	return Users{db: tx}
}

func TestCreate(t *testing.T) {
	for _, tt := range []struct {
		name, email string
		want        error
	}{
		{"another email", "ilker@example.com", nil},
		{"same email, other case", "Ayse@Example.COM", ErrEmailTaken},
	} {
		t.Run(tt.name, func(t *testing.T) {
			users := txUsers(t)
			if err := users.Create(t.Context(), "ayse@example.com"); err != nil {
				t.Fatal(err)
			}
			if err := users.Create(t.Context(), tt.email); !errors.Is(err, tt.want) {
				t.Errorf("Create(%q) error = %v, want %v", tt.email, err, tt.want)
			}
		})
	}
}

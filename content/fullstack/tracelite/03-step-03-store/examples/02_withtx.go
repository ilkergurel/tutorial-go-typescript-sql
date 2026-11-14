package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// --- Generated, abridged (example 1 has the full shape).
type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}
type Queries struct{ db DBTX }

func (q *Queries) WithTx(tx pgx.Tx) *Queries { return &Queries{db: tx} }

func (q *Queries) CreateRequirement(ctx context.Context, code string) error {
	_, err := q.db.Exec(ctx, "INSERT INTO requirements (code) VALUES ($1)", code)
	return err
}

// --- store.go, hand-written.
var ErrDuplicateCode = errors.New("duplicate code")

type Store struct {
	db *pgxpool.Pool
	q  *Queries // on the pool: for single statements, outside any transaction
}

// WithTx runs fn on Queries bound to one transaction: commit if fn returns nil, else roll back.
func (s *Store) WithTx(ctx context.Context, fn func(q *Queries) error) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error { return fn(s.q.WithTx(tx)) })
}

// storeError turns what PostgreSQL reported into the store's vocabulary.
func storeError(err error) error {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.ConstraintName == "requirements_code_key" {
		return ErrDuplicateCode
	}
	return err
}

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err == nil {
		_, err = pool.Exec(ctx, `CREATE TABLE requirements (code text CONSTRAINT requirements_code_key UNIQUE)`)
	}
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	store := &Store{db: pool, q: &Queries{db: pool}}

	// An import of three rows, the third a duplicate: all of them, or none.
	importAll := func(q *Queries) error {
		for _, code := range []string{"AUTH", "EXPORT", "AUTH"} {
			if err := q.CreateRequirement(ctx, code); err != nil {
				return err
			}
		}
		return nil
	}
	var rows int
	err = store.WithTx(ctx, func(q *Queries) error { return importAll(q) })
	pool.QueryRow(ctx, "SELECT count(*) FROM requirements").Scan(&rows)
	fmt.Printf("through q:       %v, %d rows left\n", storeError(err), rows)

	err = store.WithTx(ctx, func(q *Queries) error { return importAll(store.q) })
	pool.QueryRow(ctx, "SELECT count(*) FROM requirements").Scan(&rows)
	fmt.Printf("through store.q: %v, %d rows left\n", storeError(err), rows)
}

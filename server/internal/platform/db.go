package platform

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB is the backend's handle on the app database's pgx pool. main wraps the
// pool in it and passes it to Migrate and NewRouter; the pool's lifetime stays
// with main, which defers pool.Close so the pool outlives the HTTP drain.
// httpapi takes the concrete *DB (there is no consumer-declared interface in
// between): handleReady calls Ping, and NewRouter builds runstore and
// learningstore from Pool.
type DB struct {
	pool *pgxpool.Pool
}

func NewDB(pool *pgxpool.Pool) *DB { return &DB{pool: pool} }

// Ping proves the database answers; used by /readyz. The 2s bound makes a
// hung database fail the probe quickly instead of holding it open.
func (d *DB) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return d.pool.Ping(ctx)
}

// Pool returns the raw pool: NewRouter builds the stores from it, and Migrate
// reads its connection config to open goose's *sql.DB.
func (d *DB) Pool() *pgxpool.Pool { return d.pool }

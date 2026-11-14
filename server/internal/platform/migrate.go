// migrate.go applies embedded SQL migrations at backend startup.
//
// goose (library mode, no CLI) applies the files in filename order and records
// applied versions in its own table, so re-runs are no-ops. The files are
// embedded in the backend binary (cmd/stacklab/migrations, go:embed), matching
// the self-migrating single binary design in docs/ARCHITECTURE.md.

package platform

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Migrate applies every migration in fsys newer than the database's version.
//
// goose only looks for .sql files at the root of fsys, so the caller must pass
// a tree rooted at the migrations directory (fs.Sub away a go:embed prefix);
// otherwise goose finds nothing and fails with "no migrations found".
func Migrate(ctx context.Context, db *DB, fsys fs.FS) error {
	sdb := sql.OpenDB(stdlib.GetConnector(*db.Pool().Config().ConnConfig))
	defer func() { _ = sdb.Close() }()
	provider, err := goose.NewProvider(goose.DialectPostgres, sdb, fsys)
	if err != nil {
		return fmt.Errorf("migrate: provider: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("migrate: up: %w", err)
	}
	return nil
}

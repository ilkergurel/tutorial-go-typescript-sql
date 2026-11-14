package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type actorKey struct{} // a middleware puts the signed-in user here

// WithTx is step 3's helper with one statement added: every transaction first
// names its user, for that transaction only (is_local true).
func WithTx(ctx context.Context, db *pgxpool.Pool, fn func(pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		actor, _ := ctx.Value(actorKey{}).(string) // ActorFrom(ctx) in TraceLite
		if _, err := tx.Exec(ctx, "SELECT set_config('tracelite.actor', $1, true)", actor); err != nil {
			return err
		}
		return fn(tx)
	})
}

func main() {
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	check(err)
	cfg.MaxConns = 1 // every request below gets the same connection, as in a busy pool
	cfg.ConnConfig.OnNotice = func(_ *pgconn.PgConn, n *pgconn.Notice) { fmt.Println("   audit:", n.Message) }
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	check(err)
	defer db.Close()
	exec(ctx, db, schema)

	fmt.Println("-- WithTx for Ayşe's request, then a request that names no one")
	check(WithTx(context.WithValue(ctx, actorKey{}, "ayşe"), db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE requirements SET title = 'Kimlik doğrulama' WHERE code = 'AUTH'")
		return err
	}))
	exec(ctx, db, "UPDATE requirements SET title = 'Parola sıfırlama' WHERE code = 'AUTH-RESET'")
	var after string
	check(db.QueryRow(ctx, "SELECT format('%L', current_setting('tracelite.actor', true))").Scan(&after))
	fmt.Println("tracelite.actor after the transaction:", after)

	fmt.Println("-- SET for Ayşe's request, then a request that names no one")
	exec(ctx, db, "SET tracelite.actor = 'ayşe'")
	exec(ctx, db, "UPDATE requirements SET title = 'Kimlik doğrulama ve yetki' WHERE code = 'AUTH'")
	exec(ctx, db, "UPDATE requirements SET title = 'Parola sıfırlama e-postası' WHERE code = 'AUTH-RESET'")
}

func exec(ctx context.Context, db *pgxpool.Pool, sql string) {
	_, err := db.Exec(ctx, sql)
	check(err)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Example 1's trigger, reduced to who, what and which row. It reports through a
// notice instead of writing audit_log, so each record prints as it is made.
const schema = `CREATE TABLE requirements (code text PRIMARY KEY, title text NOT NULL);
INSERT INTO requirements VALUES ('AUTH', 'Kimlik'), ('AUTH-RESET', 'Parola');
CREATE FUNCTION audit_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE NOTICE '% % %', coalesce(nullif(current_setting('tracelite.actor', true), ''), 'db:' || session_user),
    TG_OP, NEW.code;
  RETURN NULL;
END $$;
CREATE TRIGGER requirements_audit AFTER UPDATE ON requirements
  FOR EACH ROW EXECUTE FUNCTION audit_change();`

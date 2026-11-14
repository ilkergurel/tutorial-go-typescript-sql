// Package sqlrun executes SQL lessons against the playground database, in
// three modes that all work inside a throwaway schema run_<id>:
//
//   - Run (runtime sql): BEGIN; CREATE SCHEMA; setup; the learner's SQL;
//     ROLLBACK. Nothing ever persists (an Inside StackLab lesson points here).
//   - RunNonTx (sql-nontx): autocommit, for statements that refuse to run in a
//     transaction block (VACUUM, CREATE INDEX CONCURRENTLY) and scripts with
//     their own BEGIN/COMMIT. The schema is dropped afterwards.
//   - RunMulti (sql-multisession, multisession.go): two concurrent sessions.
//
// Statements run one at a time on pgx's simple protocol, which is what psql
// uses: values arrive in Postgres' own text form, and NOTICEs (RAISE NOTICE is
// how PL/pgSQL lessons print) are attached to the statement that raised them.
// Results use the runapi wire types directly: this package exists only to
// produce them, and a second set of identical structs would just need mapping.
//
// The connection must be a non-superuser role owning only sandbox objects
// (ADR-0008): statement_timeout and lock_timeout bound every statement.
// Rejected alternative: a SAVEPOINT per statement so a failure doesn't end the
// script. psql's default (stop at the first error) is what learners will meet.
package sqlrun

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"stacklab/server/internal/runapi"
)

const (
	maxRows     = 500
	stmtTimeout = 5 * time.Second
	lockTimeout = 2 * time.Second
	runBudget   = 30 * time.Second
	// staleAfter is when a leftover run_ schema (from a crashed non-transactional
	// or multisession run) is old enough that no live run can still own it.
	staleAfter = 10 * time.Minute
	// cleanupBudget bounds teardown, which runs even after the run ctx expired.
	cleanupBudget = 5 * time.Second
)

// Request is one SQL run. Setup is the lesson's seed script; Script is the
// learner's editable SQL.
type Request struct {
	Setup  string
	Script string
}

// Session is one playground connection plus the notices the server sent on it.
type Session struct {
	conn *pgx.Conn

	mu      sync.Mutex
	notices []string
}

// Connect opens one playground connection. One connection per run (two for
// multisession) keeps transaction state isolated without a pool.
func Connect(ctx context.Context, dsn string) (*Session, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlrun: parse dsn: %w", err)
	}
	s := &Session{}
	cfg.OnNotice = func(_ *pgconn.PgConn, n *pgconn.Notice) {
		severity := n.SeverityUnlocalized
		if severity == "" {
			severity = n.Severity
		}
		s.mu.Lock()
		s.notices = append(s.notices, severity+": "+n.Message)
		s.mu.Unlock()
	}
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("sqlrun: connect: %w", err)
	}
	s.conn = conn
	return s, nil
}

// Close ends the session; any open transaction is rolled back by the server.
func (s *Session) Close(ctx context.Context) error { return s.conn.Close(ctx) }

func (s *Session) drainNotices() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.notices
	s.notices = nil
	if out == nil {
		out = []string{}
	}
	return out
}

// exec runs one statement and records its outcome. Values stay in Postgres'
// text form, as psql shows them; NULL is nil so it never looks like an empty string.
func (s *Session) exec(ctx context.Context, stmt string) runapi.StatementResult {
	if err := refuseCopyFromStdin(stmt); err != nil {
		return refused(err)
	}
	sr := runapi.StatementResult{Columns: []runapi.SQLColumn{}, Rows: [][]*string{}}
	start := time.Now()
	results, err := s.conn.PgConn().Exec(ctx, stmt).ReadAll()
	sr.DurationMs = time.Since(start).Milliseconds()
	sr.Notices = s.drainNotices()
	if err != nil {
		sr.Error = stmtError(err)
		return sr
	}
	for _, pr := range results {
		sr.Command = pr.CommandTag.String()
		sr.RowsAffected = pr.CommandTag.RowsAffected()
		sr.Columns = make([]runapi.SQLColumn, 0, len(pr.FieldDescriptions))
		for _, fd := range pr.FieldDescriptions {
			sr.Columns = append(sr.Columns, runapi.SQLColumn{Name: fd.Name, Type: oidName(fd.DataTypeOID)})
		}
		sr.Rows = make([][]*string, 0, min(len(pr.Rows), maxRows))
		for _, row := range pr.Rows[:min(len(pr.Rows), maxRows)] {
			vals := make([]*string, len(row))
			for i, v := range row {
				if v != nil {
					text := string(v)
					vals[i] = &text
				}
			}
			sr.Rows = append(sr.Rows, vals)
		}
	}
	return sr
}

// quiet runs sandbox housekeeping that is not lesson output.
func (s *Session) quiet(ctx context.Context, sql string) error {
	_, err := s.conn.PgConn().Exec(ctx, sql).ReadAll()
	s.drainNotices()
	return err
}

func stmtError(err error) *runapi.SQLStmtError {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return &runapi.SQLStmtError{
			Message: pgErr.Message, Code: pgErr.Code, Detail: pgErr.Detail,
			Hint: pgErr.Hint, Pos: pgErr.Position,
		}
	}
	return &runapi.SQLStmtError{Message: err.Error()}
}

// Run executes setup + script inside one rolled-back transaction (runtime sql).
// A failing statement ends its phase, like psql with ON_ERROR_STOP: the
// transaction is aborted, so later statements could only fail with
// "current transaction is aborted". The script may not end the transaction
// itself (guard.go): after a COMMIT, its SET LOCAL search_path would be gone
// and the rest of the script would write to public and persist.
func Run(ctx context.Context, s *Session, req Request) (*runapi.SQLResult, error) {
	if strings.TrimSpace(req.Script) == "" {
		return nil, errors.New("sqlrun: empty script")
	}
	ctx, cancel := context.WithTimeout(ctx, runBudget)
	defer cancel()

	schema := newSchemaName()
	res := &runapi.SQLResult{Schema: schema, Statements: []runapi.StatementResult{}}
	if err := s.quiet(ctx, "BEGIN; "+sandboxSQL(schema, "LOCAL")); err != nil {
		return nil, fmt.Errorf("sqlrun: sandbox schema: %w", err)
	}
	// Whatever happened, nothing persists. A cancelled ctx may have closed the
	// connection already; the server then rolls back on its own. If a
	// statement escaped the transaction anyway (runPhase stopped right after
	// it), the schema may have been committed, so it is dropped too.
	defer func() {
		cctx, ccancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupBudget)
		defer ccancel()
		escaped := s.conn.PgConn().TxStatus() == 'I'
		_ = s.quiet(cctx, "ROLLBACK")
		if escaped {
			_ = s.quiet(cctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
		}
	}()

	if runPhase(ctx, s, req.Setup, res, true) {
		runPhase(ctx, s, req.Script, res, true)
	}
	return res, nil
}

// RunNonTx executes setup + script in autocommit mode (runtime sql-nontx) and
// drops the schema afterwards. Every statement commits on its own, exactly as
// in psql, which is the point of the lessons that need this mode.
func RunNonTx(ctx context.Context, s *Session, req Request) (*runapi.SQLResult, error) {
	if strings.TrimSpace(req.Script) == "" {
		return nil, errors.New("sqlrun: empty script")
	}
	ctx, cancel := context.WithTimeout(ctx, runBudget)
	defer cancel()

	dropStaleSchemas(ctx, s)
	schema := newSchemaName()
	res := &runapi.SQLResult{Schema: schema, Statements: []runapi.StatementResult{}}
	if err := s.quiet(ctx, sandboxSQL(schema, "SESSION")); err != nil {
		return nil, fmt.Errorf("sqlrun: sandbox schema: %w", err)
	}
	defer dropSchema(ctx, s, schema)

	if runPhase(ctx, s, req.Setup, res, false) {
		runPhase(ctx, s, req.Script, res, false)
	}
	return res, nil
}

// ScratchSchema creates an empty run_ schema for a program that connects on
// its own (a Go lesson using pgx) and returns a function that drops it. The
// session must stay open until then: it does the dropping.
func ScratchSchema(ctx context.Context, s *Session) (string, func(), error) {
	dropStaleSchemas(ctx, s)
	schema := newSchemaName()
	if err := s.quiet(ctx, "CREATE SCHEMA "+schema); err != nil {
		return "", nil, fmt.Errorf("sqlrun: scratch schema: %w", err)
	}
	return schema, func() { dropSchema(ctx, s, schema) }, nil
}

// runPhase runs a multi-statement script and reports whether every statement
// succeeded. The first failure is recorded and ends the phase. inTx means the
// phase runs inside Run's sandbox transaction, which it must not end.
func runPhase(ctx context.Context, s *Session, script string, res *runapi.SQLResult, inTx bool) bool {
	for _, stmt := range splitStatements(script) {
		if inTx {
			if err := refuseTxControl(stmt); err != nil {
				res.Statements = append(res.Statements, refused(err))
				return false
			}
		}
		sr := s.exec(ctx, stmt)
		if inTx && sr.Error == nil && s.conn.PgConn().TxStatus() == 'I' {
			sr.Error = escapedSandbox
		}
		res.Statements = append(res.Statements, sr)
		if sr.Error != nil {
			return false
		}
	}
	return true
}

// sandboxSQL creates the run's schema and points the session (or, with LOCAL,
// the transaction) at it, with the per-statement limits.
func sandboxSQL(schema, scope string) string {
	return fmt.Sprintf(
		"CREATE SCHEMA %[1]s; SET %[2]s search_path TO %[1]s; SET %[2]s statement_timeout = %[3]d; SET %[2]s lock_timeout = %[4]d;",
		schema, scope, stmtTimeout.Milliseconds(), lockTimeout.Milliseconds())
}

// newSchemaName is run_<unix nanos>_<random>: the time lets dropStaleSchemas
// age leftovers, and the random part keeps two runs started in the same clock
// tick apart (Windows' wall clock is coarse).
func newSchemaName() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("run_%d_%s", time.Now().UnixNano(), hex.EncodeToString(b[:]))
}

var schemaNameRe = regexp.MustCompile(`^run_([0-9]+)_[0-9a-f]+$`)

// dropSchema removes a non-transactional run's schema. The learner's script
// may have left a transaction open (or aborted), so ROLLBACK first; outside a
// transaction it is only a warning.
func dropSchema(ctx context.Context, s *Session, schema string) {
	cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupBudget)
	defer cancel()
	_ = s.quiet(cctx, "ROLLBACK")
	_ = s.quiet(cctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
}

// dropStaleSchemas removes run_ schemas a crashed run left behind. Only
// non-transactional modes can leave them; transactional runs roll back.
// Best effort: a failure here must not fail the learner's run.
func dropStaleSchemas(ctx context.Context, s *Session) {
	results, err := s.conn.PgConn().Exec(ctx,
		"SELECT nspname FROM pg_namespace WHERE nspname LIKE 'run\\_%' AND nspowner = (SELECT oid FROM pg_roles WHERE rolname = current_user)").ReadAll()
	if err != nil || len(results) == 0 {
		return
	}
	cutoff := time.Now().Add(-staleAfter)
	for _, row := range results[0].Rows {
		name := string(row[0])
		m := schemaNameRe.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		nanos, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil || time.Unix(0, nanos).After(cutoff) {
			continue
		}
		_ = s.quiet(ctx, "DROP SCHEMA IF EXISTS "+name+" CASCADE")
	}
}

// oidName maps a Postgres type OID to its human name via pgtype; unknown
// OIDs fall back to the numeric form rather than erroring.
func oidName(oid uint32) string {
	if t, ok := typeNames.TypeForOID(oid); ok {
		return t.Name
	}
	return strconv.FormatUint(uint64(oid), 10)
}

// typeNames is shared: NewMap builds its whole type registry on every call,
// and TypeForOID only reads it (pgx v5.12), so concurrent runs can share one.
var typeNames = pgtype.NewMap()

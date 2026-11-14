// Package runexec executes one validated run request: it dispatches on the
// runtime to gorun (Go and Node programs) or sqlrun (SQL) and wraps their
// results in the shared runapi shape.
//
// Two hosts use it: the backend itself in local mode (no containers, ADR-0008)
// and cmd/runner inside the Docker sandbox (ADR-0003). One package means the two
// modes cannot drift apart. Rejected alternative: keeping this in cmd/runner,
// which would force local mode to run a second process just to reach it over
// HTTP.
package runexec

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"stacklab/server/internal/gorun"
	"stacklab/server/internal/runapi"
	"stacklab/server/internal/sqlrun"
)

const (
	// maxConcurrent bounds parallel runs: in Docker it keeps the 512 MB
	// container honest, locally it keeps a burst of runs from stalling the PC.
	maxConcurrent = 2
	// queueWait is how long a run waits for a free slot before "server busy".
	queueWait = 10 * time.Second
	// runBudget caps one whole run (compile + execute + SQL round trips).
	runBudget = 40 * time.Second
	// connectTimeout bounds opening a playground connection.
	connectTimeout = 5 * time.Second
)

// Executor runs requests under a fixed concurrency limit. Build it with New.
type Executor struct {
	playgroundDSN string
	nodeProject   string
	slots         chan struct{}
}

// New returns an Executor whose SQL runtimes connect to playgroundDSN. That
// role must not be a superuser: lesson SQL runs as it (sqlrun package doc).
// nodeProject is the app's root with node_modules, for ts-test; "" (the
// Docker runner) turns ts-test into a refusal.
func New(playgroundDSN, nodeProject string) *Executor {
	return &Executor{playgroundDSN: playgroundDSN, nodeProject: nodeProject, slots: make(chan struct{}, maxConcurrent)}
}

// Run executes req. The signature matches runclient.Client.Run so the backend
// can hold either behind one interface: a *runapi.Error is a refusal for the
// caller to pass through (busy, unsupported runtime); an error means the run
// machinery failed. Compile errors and failing lesson code are results.
func (e *Executor) Run(ctx context.Context, req runapi.Request) (*runapi.Result, *runapi.Error, error) {
	select {
	case e.slots <- struct{}{}:
		defer func() { <-e.slots }()
	case <-ctx.Done():
		return nil, busy(), nil
	case <-time.After(queueWait):
		return nil, busy(), nil
	}

	ctx, cancel := context.WithTimeout(ctx, runBudget)
	defer cancel()
	res, err := e.execute(ctx, req)
	if errors.Is(err, gorun.ErrNoNodeProject) {
		return nil, &runapi.Error{
			Status: http.StatusNotImplemented, Title: "runner error",
			Detail: "ts-test lessons run only in local mode (npm start), where the app's node_modules are available",
		}, nil
	}
	if errors.Is(err, errUnsupported) {
		// A known gap, not a broken runner: say so instead of a 502 that
		// blames the runner or the database.
		return nil, &runapi.Error{
			Status: http.StatusNotImplemented, Title: "runner error",
			Detail: "the " + string(req.Runtime) + " runtime is not implemented on the server yet",
		}, nil
	}
	if err != nil {
		return nil, nil, err
	}
	return res, nil, nil
}

func busy() *runapi.Error {
	return &runapi.Error{Status: http.StatusServiceUnavailable, Title: "runner error", Detail: "server busy"}
}

// execute dispatches on runtime. Compile errors are results (with diagnostics),
// not runner failures: the distinction is "the sandbox misbehaved" vs "the
// lesson author's code didn't compile".
func (e *Executor) execute(ctx context.Context, req runapi.Request) (*runapi.Result, error) {
	gReq := gorun.Request{
		Files:    req.Files,
		Fixtures: req.Fixtures,
		Stdin:    req.Stdin,
		TimeoutS: req.TimeoutS,
		Vet:      req.Vet,
	}
	switch req.Runtime {
	case runapi.RuntimeGo, runapi.RuntimeGoServer, runapi.RuntimeGoTest:
		if gorun.UsesPgx(req.Files) {
			env, cleanup, err := e.databaseEnv(ctx)
			if err != nil {
				return nil, err
			}
			defer cleanup()
			gReq.Env = env
		}
		if req.Runtime == runapi.RuntimeGoTest {
			return fromGorun(gorun.RunTests(ctx, gReq))
		}
		return fromGorun(gorun.Run(ctx, gReq))
	case runapi.RuntimeTSNode:
		return fromGorun(gorun.RunNode(ctx, gReq))
	case runapi.RuntimeTSTest:
		return fromGorun(gorun.RunNodeTests(ctx, gReq, e.nodeProject))
	case runapi.RuntimeSQL:
		return e.runSQL(ctx, req, sqlrun.Run)
	case runapi.RuntimeSQLNonTx:
		return e.runSQL(ctx, req, sqlrun.RunNonTx)
	case runapi.RuntimeSQLMultiSess:
		return e.runMultiSession(ctx, req)
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupported, req.Runtime)
	}
}

// errUnsupported marks a runtime execute has no case for. Run turns it into a
// refusal, so the switch above stays the only list of supported runtimes.
var errUnsupported = errors.New("unsupported runtime")

// firstFile pulls the learner's script out of the files map; SQL lessons ship
// one editable .sql file per example.
func firstFile(req runapi.Request) string {
	for _, body := range req.Files {
		return body
	}
	return ""
}

func (e *Executor) connect(ctx context.Context) (*sqlrun.Session, error) {
	if e.playgroundDSN == "" {
		return nil, errors.New("runner database url not configured")
	}
	connCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	return sqlrun.Connect(connCtx, e.playgroundDSN)
}

// databaseEnv gives a Go lesson that imports pgx an empty schema of its own in
// the playground database, passed as DATABASE_URL with search_path set, the
// way a real service gets its connection string. The program connects by
// itself, so whatever it commits stays inside that schema, which is dropped
// after the run. Rejected alternative: handing it a connection from here,
// which would hide the very pool and connect code the go.postgres lessons teach.
func (e *Executor) databaseEnv(ctx context.Context) ([]string, func(), error) {
	s, err := e.connect(ctx)
	if err != nil {
		return nil, nil, err
	}
	closeSession := func() { _ = s.Close(context.WithoutCancel(ctx)) }
	schema, drop, err := sqlrun.ScratchSchema(ctx, s)
	if err != nil {
		closeSession()
		return nil, nil, err
	}
	dsn, err := withSearchPath(e.playgroundDSN, schema)
	if err != nil {
		drop()
		closeSession()
		return nil, nil, err
	}
	return []string{"DATABASE_URL=" + dsn}, func() { drop(); closeSession() }, nil
}

// withSearchPath adds search_path to a URL or keyword/value DSN. pgx sends
// parameters it doesn't know as session settings, so the program's every
// connection starts in the run's schema.
func withSearchPath(dsn, schema string) (string, error) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return "", fmt.Errorf("runexec: playground dsn: %w", err)
		}
		q := u.Query()
		q.Set("search_path", schema)
		u.RawQuery = q.Encode()
		return u.String(), nil
	}
	return dsn + " search_path=" + schema, nil
}

// sqlMode is sqlrun.Run or sqlrun.RunNonTx.
type sqlMode func(context.Context, *sqlrun.Session, sqlrun.Request) (*runapi.SQLResult, error)

func (e *Executor) runSQL(ctx context.Context, req runapi.Request, mode sqlMode) (*runapi.Result, error) {
	s, err := e.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = s.Close(context.WithoutCancel(ctx)) }()

	start := time.Now()
	res, err := mode(ctx, s, sqlrun.Request{Setup: req.Setup, Script: firstFile(req)})
	if err != nil {
		return nil, err
	}
	// ExitCode stays 0 even when a statement failed: for SQL the error is a
	// result (many lessons end on a deliberate constraint violation), and the
	// text in Stdout, which the expected files compare against, carries it.
	return &runapi.Result{
		Stdout:     sqlrun.Text(res.Statements),
		DurationMs: time.Since(start).Milliseconds(),
		SQL:        res,
	}, nil
}

// runMultiSession opens TWO playground connections (session A and B).
func (e *Executor) runMultiSession(ctx context.Context, req runapi.Request) (*runapi.Result, error) {
	a, err := e.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = a.Close(context.WithoutCancel(ctx)) }()
	b, err := e.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = b.Close(context.WithoutCancel(ctx)) }()

	start := time.Now()
	multi, err := sqlrun.RunMulti(ctx, a, b, req.Setup, firstFile(req))
	if err != nil {
		return nil, err
	}
	// As in runSQL, failures are results: a deadlock or a serialization
	// failure is usually the point of the lesson.
	return &runapi.Result{
		Stdout:     multi.Text(),
		DurationMs: time.Since(start).Milliseconds(),
		SQL:        &runapi.SQLResult{Schema: multi.Schema, Statements: multi.Setup},
		Steps:      multi.Steps,
	}, nil
}

func fromGorun(res gorun.Result, err error) (*runapi.Result, error) {
	if err != nil {
		return nil, err
	}
	out := &runapi.Result{
		Stdout:     res.Stdout,
		Stderr:     res.Stderr,
		ExitCode:   res.ExitCode,
		DurationMs: res.DurationMs,
		TimedOut:   res.TimedOut,
		Truncated:  res.Truncated,
	}
	for _, d := range res.Diagnostics {
		out.Diagnostics = append(out.Diagnostics, runapi.Diagnostic{File: d.File, Line: d.Line, Col: d.Col, Message: d.Message})
	}
	for _, f := range res.Files {
		out.Files = append(out.Files, runapi.WrittenFile{Path: f.Path, Size: f.Size, Preview: f.Preview})
	}
	return out, nil
}

// Multisession mode: a script whose steps are tagged A: or B: runs on two
// connections, like two psql terminals side by side. Session state (open
// transactions, locks, SETs) persists per session across steps, which is the
// whole point: isolation and locking lessons need one session to hold a
// transaction open while the other acts.
//
// The sessions run concurrently. Steps are sent in script order; a step that
// has not finished within blockWindow is reported as "waiting" (usually
// blocked on the other session's lock) and the next step is sent anyway, just
// as you would switch terminals. Its result appears when it arrives, e.g.
// after the other session commits, or when the deadlock detector picks it.
// Rejected alternative: strictly sequential steps with a short lock_timeout.
// Simpler, but "B waits until A commits" would show up as an error, and a
// deadlock could never form: the two lessons that matter most would lie.
//
// Setup runs first and commits, so both sessions see its tables; everything
// lives in a run_ schema that is dropped afterwards.
package sqlrun

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"stacklab/server/internal/runapi"
)

const (
	// blockWindow is how long a step may run before it counts as waiting and
	// the next step is sent. It stays well under the server's deadlock_timeout
	// (1s by default, and only a superuser may change it), so in a deadlock
	// the second session is already waiting when the first one's deadlock
	// check runs: the same session is the victim on every run.
	blockWindow = 400 * time.Millisecond
	// settleWindow is how long to wait, after a step returns, for a statement
	// it unblocked in the other session: those finish in milliseconds.
	settleWindow = 100 * time.Millisecond
	// multiLockTimeout is longer than single-session mode's: here a lock wait
	// is the lesson, and the other session usually releases it a step later.
	multiLockTimeout = 4 * time.Second
)

// MultiStep is one parsed step: a statement and the session that runs it.
type MultiStep struct {
	Session byte // 'A' or 'B'
	SQL     string
}

// MultiResult is a two-session run. Steps are in script order; Text tells
// the story in the order things happened.
type MultiResult struct {
	Schema string
	Setup  []runapi.StatementResult
	Steps  []runapi.StepResult
	events []event
}

type event struct {
	step    int  // index into Steps
	waiting bool // still running when the next step was sent
}

// parseSessionScript splits a tagged script into steps. A step runs from its
// tag ("A:" or "B:" at the start of a line) to the next tag; untagged lines
// before the first tag belong to session A. Write one statement per step.
// Comment and blank lines at the end of a step describe the next one, so they
// are dropped, and a step without code is no step at all.
func parseSessionScript(script string) []MultiStep {
	var steps []MultiStep
	var cur *MultiStep
	var lines []string
	flush := func() {
		for len(lines) > 0 {
			last := strings.TrimSpace(lines[len(lines)-1])
			if last != "" && !strings.HasPrefix(last, "--") {
				break
			}
			lines = lines[:len(lines)-1]
		}
		if cur != nil {
			cur.SQL = strings.TrimSpace(strings.Join(lines, "\n"))
			if hasCode(cur.SQL) {
				steps = append(steps, *cur)
			}
		}
		cur, lines = nil, nil
	}
	for _, line := range strings.Split(strings.ReplaceAll(script, "\r\n", "\n"), "\n") {
		if session, rest, ok := sessionTag(line); ok {
			flush()
			cur = &MultiStep{Session: session}
			line = rest
		}
		if cur == nil {
			cur = &MultiStep{Session: 'A'}
		}
		lines = append(lines, line)
	}
	flush()
	return steps
}

// sessionTag recognizes "A:" / "B:" at the start of a line (after
// indentation), followed by whitespace or the end of the line.
func sessionTag(line string) (byte, string, bool) {
	t := strings.TrimLeft(line, " \t")
	if len(t) < 2 || (t[0] != 'A' && t[0] != 'B') || t[1] != ':' {
		return 0, "", false
	}
	if len(t) > 2 && t[2] != ' ' && t[2] != '\t' {
		return 0, "", false
	}
	return t[0], strings.TrimLeft(t[2:], " \t"), true
}

// RunMulti executes setup, then the tagged script on sessions a and b.
func RunMulti(ctx context.Context, a, b *Session, setup, script string) (*MultiResult, error) {
	steps := parseSessionScript(script)
	if len(steps) == 0 {
		return nil, errors.New("sqlrun: empty multisession script")
	}
	ctx, cancel := context.WithTimeout(ctx, runBudget)
	defer cancel()

	dropStaleSchemas(ctx, a)
	schema := newSchemaName()
	if err := a.quiet(ctx, "CREATE SCHEMA "+schema); err != nil {
		return nil, fmt.Errorf("sqlrun: sandbox schema: %w", err)
	}
	// Roll B back first: it may hold locks that DROP SCHEMA would wait on.
	defer func() {
		cctx, ccancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupBudget)
		defer ccancel()
		_ = b.quiet(cctx, "ROLLBACK")
		dropSchema(cctx, a, schema)
	}()
	settings := fmt.Sprintf("SET search_path TO %s; SET statement_timeout = %d; SET lock_timeout = %d;",
		schema, stmtTimeout.Milliseconds(), multiLockTimeout.Milliseconds())
	for _, s := range []*Session{a, b} {
		if err := s.quiet(ctx, settings); err != nil {
			return nil, fmt.Errorf("sqlrun: session settings: %w", err)
		}
	}

	res := &MultiResult{Schema: schema, Steps: make([]runapi.StepResult, len(steps))}
	setupRes := &runapi.SQLResult{Statements: []runapi.StatementResult{}}
	ok := runPhase(ctx, a, setup, setupRes, false) // autocommit, like RunNonTx
	res.Setup = setupRes.Statements
	for i, st := range steps {
		res.Steps[i] = runapi.StepResult{Step: i + 1, Session: string(st.Session), SQL: st.SQL}
	}
	if !ok {
		return res, nil // the setup error is the result; the steps never ran
	}

	runSteps(ctx, map[byte]*Session{'A': a, 'B': b}, steps, res)
	return res, nil
}

// runSteps drives the two sessions so that the transcript is the same on every
// run, even where two results race over the network (A's COMMIT reply and the
// UPDATE it unblocked in B can arrive in either order):
//
//   - a session gets its next step only once its previous one returned, as in
//     a terminal; meanwhile the other session's results are recorded as they
//     come;
//   - while a step runs, results from the other session are held back and
//     recorded after it: what you just typed answers first;
//   - after a step returns, a short settle window lets a statement it
//     unblocked finish, so cause and effect print next to each other.
//
// Every exec is bounded by statement_timeout, lock_timeout and ctx, so waiting
// for a busy session always ends.
func runSteps(ctx context.Context, sessions map[byte]*Session, steps []MultiStep, res *MultiResult) {
	type done struct {
		idx int
		sr  runapi.StatementResult
	}
	finished := make(chan done, len(steps))
	running := map[byte]int{} // session -> index of its unfinished step
	order := 0
	record := func(d done) {
		order++
		res.Steps[d.idx].StatementResult = d.sr
		res.Steps[d.idx].FinishedOrder = order
		res.events = append(res.events, event{step: d.idx})
		if running[steps[d.idx].Session] == d.idx {
			delete(running, steps[d.idx].Session)
		}
	}
	settle := func() {
		if len(running) == 0 {
			return
		}
		timer := time.NewTimer(settleWindow)
		defer timer.Stop()
		for len(running) > 0 {
			select {
			case d := <-finished:
				record(d)
			case <-timer.C:
				return
			}
		}
	}

	for i, st := range steps {
		if _, busy := running[st.Session]; busy {
			for {
				if _, busy := running[st.Session]; !busy {
					break
				}
				record(<-finished)
			}
			settle()
		}

		running[st.Session] = i
		go func(s *Session, i int) { finished <- done{i, s.exec(ctx, steps[i].SQL)} }(sessions[st.Session], i)

		timer := time.NewTimer(blockWindow)
		var others []done
	wait:
		for {
			select {
			case d := <-finished:
				if d.idx != i {
					others = append(others, d)
					continue
				}
				record(d)
				for _, o := range others {
					record(o)
				}
				settle()
				break wait
			case <-timer.C:
				res.Steps[i].Waited = true
				res.events = append(res.events, event{step: i, waiting: true})
				for _, o := range others {
					record(o)
				}
				break wait
			}
		}
		timer.Stop()
	}
	for len(running) > 0 {
		record(<-finished)
	}
}

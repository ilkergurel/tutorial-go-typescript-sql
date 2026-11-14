// Package gorun executes lesson programs as local processes in a throwaway
// working directory: Go (build, then run), `go test` (runtime go-test) and
// Node for TypeScript (runtime ts-node). Node lives here too because the
// process hygiene is the whole job and is identical; a second package would
// copy it.
//
// Threat model (ADR-0003): this protects the owner from their own mistakes —
// infinite loops, huge output, runaway processes — not from a determined attacker.
// In Docker mode the container (no network, read-only rootfs, cpu/mem/pids
// limits) is the real boundary. In local mode (ADR-0008) this package's per-run
// hygiene is all there is: fresh temp dir, compile/run timeouts, output caps,
// and killing the whole process tree when a run or a build step ends, whether
// by timeout, cancellation or a normal exit (proc_unix.go, proc_windows.go),
// and, on Windows and for the direct child on Linux, when the backend itself
// dies (proc_windows.go, proc_linux.go).
//
// Rejected alternative: running `go run` — it conflates compile and run timeouts
// and hides compile errors from the diagnostics parser. We build, then run.
package gorun

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	compileTimeout = 20 * time.Second
	defaultRunTTL  = 5 * time.Second
	maxRunTTL      = 15 * time.Second
	// testTimeoutGrace is how much later than go test's own -timeout the
	// outer kill comes: time for the timed-out test binary to print its
	// goroutine dump and for go test to report FAIL, so the learner reads
	// Go's "test timed out" panic instead of a bare kill.
	testTimeoutGrace = 3 * time.Second
	// pipeGrace bounds how long a run keeps reading output once the program
	// has exited and its process tree is dead: a process that left the tree
	// and still holds stdout open must not hang the run forever.
	pipeGrace = 2 * time.Second
	// binName is the compiled lesson program inside the work dir.
	binName = "lesson-bin" + exeSuffix
	// OutputCap per stream, with a visible truncation marker (spec §6.1).
	OutputCap = 64 * 1024
)

// Diagnostics are compiler/vet errors mapped to file/line/col for editor markers.
type Diagnostic struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	Message string `json:"message"`
}

// Result is the single typed shape every runtime returns (spec §6.1). The TS
// side mirrors it as a discriminated union on TimedOut/ExitCode.
type Result struct {
	Stdout      string        `json:"stdout"`
	Stderr      string        `json:"stderr"`
	ExitCode    int           `json:"exitCode"`
	DurationMs  int64         `json:"durationMs"`
	TimedOut    bool          `json:"timedOut"`
	Truncated   bool          `json:"truncated"`
	Diagnostics []Diagnostic  `json:"diagnostics"`
	Files       []WrittenFile `json:"files"`
}

// WrittenFile lists files the program created or changed (relative to the work
// dir), with a short preview, for the "writing files" lessons.
type WrittenFile struct {
	Path    string `json:"path"`
	Size    int    `json:"size"`
	Preview string `json:"preview"`
}

// Request describes one run. Files maps names to contents; Fixtures are
// pre-seeded files addressed relative to the work dir (e.g. "app.log").
type Request struct {
	Files    map[string]string
	Fixtures map[string]string
	Stdin    string
	TimeoutS int
	Vet      bool
	// Env adds KEY=value pairs to the program's environment (DATABASE_URL for
	// lessons that use pgx); the backend's own environment is inherited.
	Env []string
}

// compilerError matches "./main.go:12:5: message" (and file-only variants).
var compilerError = regexp.MustCompile(`^(\S+\.go):(\d+):(?:(\d+):)?\s*(.*)$`)

// goScaffold is the minimal module `go build` needs with GOPROXY=off for a
// standard-library-only lesson.
var goScaffold = map[string]string{"go.mod": "module lesson\n\ngo 1.27\n"}

// pgxScaffold is the module for lessons that import pgx: the same pgx and
// golang.org/x versions the backend uses, so `go build` resolves everything
// from the module cache the backend's own build filled, with no network. The
// files were produced by `go mod tidy`; TestPgxScaffoldMatchesServer keeps the
// pgx version in step with server/go.mod. Rejected alternative: a go.work or
// replace directive into the backend module, which would expose its internals
// to lesson code.
var pgxScaffold = map[string]string{"go.mod": pgxGoMod, "go.sum": pgxGoSum}

//go:embed pgxlesson.go.mod
var pgxGoMod string

//go:embed pgxlesson.go.sum
var pgxGoSum string

// UsesPgx reports whether a lesson's Go files import pgx. Such lessons get
// pgxScaffold here and, from runexec, a database to talk to.
func UsesPgx(files map[string]string) bool {
	return importsAny(files, `"github.com/jackc/pgx/v5`)
}

// pgxScaffold also lists golang.org/x/sync and golang.org/x/text, so lessons
// can use errgroup or x/text/cases (full Turkish case rules) offline too.
func goScaffoldFor(req Request) map[string]string {
	if importsAny(req.Files, `"github.com/jackc/pgx/v5`, `"golang.org/x/sync/`, `"golang.org/x/text/`) {
		return pgxScaffold
	}
	return goScaffold
}

func importsAny(files map[string]string, paths ...string) bool {
	for name, src := range files {
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		for _, p := range paths {
			if strings.Contains(src, p) {
				return true
			}
		}
	}
	return false
}

// withEnv returns the environment for a lesson program: inherited plus extra.
func withEnv(base []string, extra []string) []string {
	if len(extra) == 0 {
		return base
	}
	if base == nil {
		base = os.Environ()
	}
	return append(base, extra...)
}

// nodeScaffold makes .ts and .js files ES modules, so import/export and
// top-level await work without depending on Node's module-type detection.
var nodeScaffold = map[string]string{"package.json": "{\"type\": \"module\"}\n"}

// Run compiles and executes the request in a fresh temp directory.
func Run(ctx context.Context, req Request) (Result, error) {
	scaffold := goScaffoldFor(req)
	workDir, err := newWorkDir(req, scaffold)
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(workDir)

	bin := filepath.Join(workDir, binName)
	if build := goBuild(ctx, workDir, bin); build.ExitCode != 0 {
		// A compile error is a result, not a runner failure, and so is a go
		// that can't start or a build that runs out of time.
		out := build.Stdout + build.Stderr
		res := Result{
			Diagnostics: parseDiagnostics(out),
			Stderr:      truncate(out),
			ExitCode:    1,
			DurationMs:  build.DurationMs,
			Files:       []WrittenFile{},
		}
		if build.TimedOut {
			res.TimedOut, res.ExitCode = true, -1
		}
		return res, nil
	}
	var vetDiags []Diagnostic
	vetNote := ""
	if req.Vet {
		vet := goVet(ctx, workDir)
		out := vet.Stdout + vet.Stderr
		vetDiags = parseDiagnostics(out)
		if vet.ExitCode != 0 && len(vetDiags) == 0 {
			// vet failed without a finding to mark (it couldn't start, ran out
			// of time or crashed): say so rather than pass for a clean vet.
			vetNote = "\ngorun: go vet failed:\n" + truncate(out)
		}
	}

	res := runProcess(ctx, workDir, runTTL(req), req.Stdin, withEnv(nil, req.Env), bin)
	res.Diagnostics = append(res.Diagnostics, vetDiags...)
	if vetNote != "" {
		res.Stderr = strings.TrimSpace(res.Stderr + vetNote)
	}
	res.Files = listWritten(workDir, req, scaffold)
	return res, nil
}

// RunTests executes `go test -v` over the seeded files (go-test runtime).
// GOPROXY=off keeps the sandbox hermetic (stdlib, or pgx from the module
// cache). Everything the test process prints lands in stdout; the go test
// harness's own lines (RUN/PASS/FAIL) make the output self-explanatory.
//
// The build gets compileTimeout, as in Run, and the tests testTTL
// (timeout_s, 15 s without it). go test builds and runs in one command, so
// the budget reaches the test binary as -timeout: a hanging test ends in
// Go's own "test timed out" panic, which names the running tests and dumps
// every goroutine, and go test reports FAIL. The outer kill (goTestLimits) comes later and
// catches what -timeout can't see: a build that hangs, or a TestMain that
// blocks before m.Run starts the clock.
func RunTests(ctx context.Context, req Request) (Result, error) {
	scaffold := goScaffoldFor(req)
	workDir, err := newWorkDir(req, scaffold)
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(workDir)

	testTimeout, total := goTestLimits(req)
	env := withEnv(append(os.Environ(), "GOPROXY=off", "GOFLAGS=-mod=mod"), req.Env)
	res := runProcess(ctx, workDir, total, "", env, "go", "test", "-v", "-timeout="+testTimeout.String(), "./...")
	res.Files = listWritten(workDir, req, scaffold)
	return res, nil
}

// goTestLimits returns go test's -timeout, the tests' run budget, and the
// deadline for the whole go test command: the build allowance, the budget
// and testTimeoutGrace, so Go's own timeout always fires first. A build that
// takes longer than compileTimeout eats into that grace and then into the
// budget instead of failing at once, as Run's separate build would.
func goTestLimits(req Request) (testTimeout, total time.Duration) {
	testTimeout = testTTL(req)
	return testTimeout, compileTimeout + testTimeout + testTimeoutGrace
}

// RunNode executes a TypeScript (or JavaScript) entry file with Node's type
// stripping (runtime ts-node). Stripping only erases annotations, so lessons
// are limited to erasable syntax: no enums, namespaces or parameter
// properties, which is the subject of the ts.node lessons. Nothing is type
// checked; ts-browser has the same limit (esbuild).
//
// --experimental-strip-types is the default from Node 23.6 and harmless there;
// Node 22 needs it. --disable-warning keeps Node 22's "experimental" notice
// out of the learner's stderr. Verified on Node 22.15.1 and 24.22.0.
func RunNode(ctx context.Context, req Request) (Result, error) {
	entry, err := nodeEntry(req.Files)
	if err != nil {
		return Result{}, err
	}
	workDir, err := newWorkDir(req, nodeScaffold)
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(workDir)

	res := runProcess(ctx, workDir, runTTL(req), req.Stdin, nil,
		"node", "--experimental-strip-types", "--disable-warning=ExperimentalWarning", entry)
	res.Files = listWritten(workDir, req, nodeScaffold)
	return res, nil
}

// nodeEntry picks the file Node starts: main.ts if present, else the only
// .ts/.mts/.js/.mjs file. Other files are modules it imports.
func nodeEntry(files map[string]string) (string, error) {
	if _, ok := files["main.ts"]; ok {
		return "main.ts", nil
	}
	var candidates []string
	for name := range files {
		switch filepath.Ext(name) {
		case ".ts", ".mts", ".js", ".mjs":
			candidates = append(candidates, name)
		}
	}
	if len(candidates) != 1 {
		slices.Sort(candidates)
		return "", fmt.Errorf("gorun: ts-node needs main.ts or exactly one script, got %q", candidates)
	}
	return candidates[0], nil
}

func runTTL(req Request) time.Duration {
	if req.TimeoutS > 0 {
		return min(time.Duration(req.TimeoutS)*time.Second, maxRunTTL)
	}
	return defaultRunTTL
}

// testTTL is the run budget of the test runtimes (go-test, ts-test).
// timeout_s works as in the go runtime, up to the same maxRunTTL. Without it
// the run gets that maximum, not the go runtime's 5 s default: a test run
// pays for start-up a lesson program doesn't have (Node, Vitest and jsdom,
// or Edge for Playwright; the go test harness) and often runs a whole table
// of cases. For go-test the build comes on top (goTestLimits).
func testTTL(req Request) time.Duration {
	if req.TimeoutS > 0 {
		return runTTL(req)
	}
	return maxRunTTL
}

// newWorkDir creates a fresh temp dir holding the request's files, fixtures
// and the runtime's scaffold (learner files win over scaffold files).
func newWorkDir(req Request, scaffold map[string]string) (string, error) {
	return newWorkDirIn("", req, scaffold)
}

// newWorkDirIn is newWorkDir under base ("" = the system temp dir).
func newWorkDirIn(base string, req Request, scaffold map[string]string) (string, error) {
	if len(req.Files) == 0 {
		return "", errors.New("gorun: no files to run")
	}
	workDir, err := os.MkdirTemp(base, "stacklab-run-")
	if err != nil {
		return "", fmt.Errorf("gorun: temp dir: %w", err)
	}
	if err := seed(workDir, req, scaffold); err != nil {
		os.RemoveAll(workDir)
		return "", err
	}
	return workDir, nil
}

// runProcess runs one program under the shared hygiene: a deadline, output
// capped per stream, and the whole process tree killed when the run ends,
// however it ends: the deadline passes, the request is cancelled, or the
// program exits and leaves processes behind (a lesson's background child,
// Vitest workers, a browser). env nil means the backend's environment.
//
// gorun owns the output pipes instead of handing exec an io.Writer. With a
// Writer, Wait keeps copying until every process holding the pipe has exited,
// so a leftover child could be killed only after Wait gave up (pipeGrace).
// With *os.File, Wait returns as soon as the program exits; the tree is killed
// right then, and the copies reach EOF.
func runProcess(ctx context.Context, dir string, ttl time.Duration, stdin string, env []string, name string, args ...string) Result {
	runCtx, cancel := context.WithTimeout(ctx, ttl)
	defer cancel()

	cmd := exec.CommandContext(runCtx, name, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin = strings.NewReader(stdin)
	tree := newProcessTree(cmd) // also sets cmd.Cancel: deadline or cancel kills the tree
	defer tree.release()
	// After cmd.Cancel, how long Wait waits for the program to die before
	// killing it directly, and how long it may feed stdin to a leftover child.
	cmd.WaitDelay = pipeGrace

	stdout, stdoutW, err := startCapture()
	if err != nil {
		return failedRun(err)
	}
	stderr, stderrW, err := startCapture()
	if err != nil {
		stdoutW.Close()
		stdout.finish(context.Background()) // EOF at once: no one else has the write end
		return failedRun(err)
	}
	cmd.Stdout, cmd.Stderr = stdoutW, stderrW

	start := time.Now()
	runErr := cmd.Start()
	// The program holds its own copies of the write ends now. Ours must go,
	// or the copies would never see EOF.
	stdoutW.Close()
	stderrW.Close()
	if runErr == nil {
		tree.started()
		runErr = cmd.Wait()
		tree.killLeftovers()
	}
	// Read before draining, which may take up to pipeGrace more.
	timedOut := errors.Is(runCtx.Err(), context.DeadlineExceeded)
	// The tree is dead, so EOF comes at once, unless a process left it on
	// purpose (setsid on Unix): then stop reading after pipeGrace.
	grace, stopGrace := context.WithTimeout(context.Background(), pipeGrace)
	allRead := stdout.finish(grace)
	allRead = stderr.finish(grace) && allRead
	stopGrace()

	res := Result{
		DurationMs:  time.Since(start).Milliseconds(),
		Stdout:      stdout.buf.String(),
		Stderr:      stderr.buf.String(),
		Truncated:   stdout.buf.truncated || stderr.buf.truncated,
		Diagnostics: []Diagnostic{},
		Files:       []WrittenFile{},
	}
	if !allRead {
		res.Stderr = strings.TrimSpace(res.Stderr + "\ngorun: stopped reading output " + pipeGrace.String() +
			" after the run ended: a process outside the program's process tree still holds it")
	}
	switch {
	case timedOut:
		res.TimedOut = true
		res.ExitCode = -1
	case runErr != nil:
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.ExitCode = -1
			res.Stderr = strings.TrimSpace(res.Stderr + "\ngorun: " + runErr.Error())
		}
	}
	return res
}

// failedRun is the result when the run's plumbing could not be set up.
func failedRun(err error) Result {
	return Result{ExitCode: -1, Stderr: err.Error(), Diagnostics: []Diagnostic{}, Files: []WrittenFile{}}
}

// capture drains one of the program's output streams into a cappedBuffer.
type capture struct {
	buf  cappedBuffer
	r    *os.File
	done chan struct{} // closed when the copy has stopped
}

// startCapture returns a capture that is already reading, and the pipe's
// write end to hand the program.
func startCapture() (*capture, *os.File, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, nil, fmt.Errorf("gorun: output pipe: %w", err)
	}
	c := &capture{buf: cappedBuffer{cap: OutputCap}, r: r, done: make(chan struct{})}
	go func() {
		defer close(c.done)
		_, _ = io.Copy(&c.buf, r)
	}()
	return c, w, nil
}

// finish waits for the copy to reach EOF or for grace to end, then closes
// the read end, which stops a copy that is still blocked in Read (on Windows
// too: closing a pipe cancels the pending read, which exec's WaitDelay relies
// on as well). It reports whether EOF came first.
func (c *capture) finish(grace context.Context) bool {
	eof := true
	select {
	case <-c.done:
	case <-grace.Done():
		eof = false
	}
	c.r.Close()
	<-c.done
	return eof
}

// goBuild compiles the lesson program; a non-zero ExitCode is a failed build.
func goBuild(ctx context.Context, dir, bin string) Result {
	return goTool(ctx, dir, append(os.Environ(), "GOPROXY=off", "GOFLAGS=-mod=mod"), "build", "-o", bin, ".")
}

// goVet vets the lesson package; a non-zero ExitCode means findings or a vet
// that couldn't run.
func goVet(ctx context.Context, dir string) Result {
	return goTool(ctx, dir, append(os.Environ(), "GOPROXY=off"), "vet", "./...")
}

// goTool runs one go command of the build step through runProcess, with the
// lesson program's hygiene: a timeout or a cancelled request ends the
// compiler, linker or vet tool that go started too (exec's default Cancel
// would kill go alone and leave them running), and a go that can't start
// says why in Stderr (runProcess adds `gorun: exec: "go": ...`) instead of
// failing with empty output. go prints its errors on stderr.
func goTool(ctx context.Context, dir string, env []string, args ...string) Result {
	res := runProcess(ctx, dir, compileTimeout, "", env, "go", args...)
	if res.TimedOut {
		res.Stderr = strings.TrimSpace(res.Stderr + "\ngorun: go " + args[0] + " timed out (limit " + compileTimeout.String() + ")")
	}
	return res
}

func seed(workDir string, req Request, scaffold map[string]string) error {
	for name, content := range scaffold {
		if _, own := req.Files[name]; own {
			continue
		}
		if err := os.WriteFile(filepath.Join(workDir, name), []byte(content), 0o600); err != nil {
			return fmt.Errorf("gorun: scaffold %q: %w", name, err)
		}
	}
	for name, content := range req.Files {
		// Reject path escapes: names must stay inside the work dir. IsLocal
		// rather than a ".."/IsAbs check: on Windows "/x" and "C:x" are not
		// absolute yet still escape, and names like "NUL" are devices.
		if !filepath.IsLocal(name) {
			return fmt.Errorf("gorun: illegal file name %q", name)
		}
		if err := writeFile(workDir, name, content); err != nil {
			return fmt.Errorf("gorun: seed %q: %w", name, err)
		}
	}
	for name, content := range req.Fixtures {
		if !filepath.IsLocal(name) {
			return fmt.Errorf("gorun: illegal fixture name %q", name)
		}
		if err := writeFile(workDir, name, content); err != nil {
			return fmt.Errorf("gorun: fixture %q: %w", name, err)
		}
	}
	return nil
}

func writeFile(workDir, name, content string) error {
	p := filepath.Join(workDir, filepath.Clean(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(content), 0o600)
}

// listWritten reports files in the work dir that weren't part of the seed,
// with a short text preview.
func listWritten(workDir string, req Request, scaffold map[string]string) []WrittenFile {
	seeded := map[string]bool{binName: true}
	for name := range scaffold {
		seeded[name] = true
	}
	for name := range req.Files {
		seeded[filepath.Clean(name)] = true
	}
	for name := range req.Fixtures {
		seeded[filepath.Clean(name)] = true
	}
	var files []WrittenFile
	entries, err := os.ReadDir(workDir)
	if err != nil {
		return files
	}
	for _, e := range entries {
		if e.IsDir() || seeded[e.Name()] {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		preview := ""
		if info.Size() <= 4096 {
			if b, err := os.ReadFile(filepath.Join(workDir, e.Name())); err == nil {
				preview = string(bytes.Split(b, []byte("\n"))[0])
				if len(preview) > 120 {
					preview = preview[:120] + "…"
				}
			}
		}
		files = append(files, WrittenFile{Path: e.Name(), Size: int(info.Size()), Preview: preview})
	}
	return files
}

func parseDiagnostics(out string) []Diagnostic {
	var diags []Diagnostic
	for _, ln := range strings.Split(out, "\n") {
		if m := compilerError.FindStringSubmatch(ln); m != nil {
			n, _ := strconv.Atoi(m[2])
			col := 0
			if m[3] != "" {
				fmt.Sscanf(m[3], "%d", &col)
			}
			diags = append(diags, Diagnostic{File: m[1], Line: n, Col: col, Message: m[4]})
		}
	}
	if diags == nil {
		diags = []Diagnostic{}
	}
	return diags
}

// truncate caps compiler output the way cappedBuffer caps a running program's.
func truncate(s string) string {
	c := cappedBuffer{cap: OutputCap}
	_, _ = c.Write([]byte(s))
	return c.String()
}

// cappedBuffer caps at cap bytes and flags truncation; the output pipes drain
// into it while the program runs.
type cappedBuffer struct {
	buf       bytes.Buffer
	cap       int
	truncated bool
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	if c.truncated {
		// Discard, but report success: an error would stop the copy, the pipe
		// would fill, and the program would block on its next write.
		return len(p), nil
	}
	room := c.cap - c.buf.Len()
	if len(p) <= room {
		return c.buf.Write(p)
	}
	c.buf.Write(p[:max(room, 0)])
	// Cut on a character boundary: half a multi-byte character (Turkish ğ, ş,
	// İ are two bytes) would reach the browser as U+FFFD, because the JSON
	// encoder replaces invalid UTF-8. The partial character may have started
	// in an earlier Write, so the cut looks at what is already buffered.
	c.buf.Truncate(c.buf.Len() - partialRuneLen(c.buf.Bytes()))
	c.buf.WriteString("\n[output truncated]")
	c.truncated = true
	return len(p), nil
}

func (c *cappedBuffer) String() string { return c.buf.String() }

// partialRuneLen returns how many bytes at the end of b are an unfinished
// UTF-8 sequence, 0 when b ends on a whole character. Only the last
// utf8.UTFMax-1 bytes can hold one. Invalid bytes count as whole characters,
// as utf8.FullRune treats them, so binary output is cut where it is.
func partialRuneLen(b []byte) int {
	for i := len(b) - 1; i >= 0 && i >= len(b)-(utf8.UTFMax-1); i-- {
		if utf8.RuneStart(b[i]) {
			if utf8.FullRune(b[i:]) {
				return 0
			}
			return len(b) - i
		}
	}
	return 0
}

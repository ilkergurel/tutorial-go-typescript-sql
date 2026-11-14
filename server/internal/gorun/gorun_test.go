// gorun_test.go exercises the sandbox hygiene guarantees against the real Go
// toolchain and OS: output caps, the process-tree kill on timeout, on cancel
// and after a normal exit, and the path-escape checks in seed().
// Build-tagged "integration" because the tests compile real programs or start
// real process trees (~seconds; most need go on PATH).
//
//go:build integration

package gorun

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func hello(files map[string]string) Request {
	return Request{Files: files}
}

const helloMain = `package main

import "fmt"

func main() { fmt.Println("hello stacklab") }
`

func TestRunHappyPath(t *testing.T) {
	res, err := Run(context.Background(), hello(map[string]string{"main.go": helloMain}))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.ExitCode != 0 || !strings.Contains(res.Stdout, "hello stacklab") {
		t.Fatalf("unexpected result: %+v", res)
	}
	if res.TimedOut || res.Truncated || len(res.Diagnostics) != 0 {
		t.Fatalf("clean run polluted: %+v", res)
	}
}

func TestRunCompileErrorIsResult(t *testing.T) {
	bad := "package main\n\nfunc main() { undefined() }\n"
	res, err := Run(context.Background(), hello(map[string]string{"main.go": bad}))
	if err != nil {
		t.Fatalf("compile error must not be a runner failure: %v", err)
	}
	if res.ExitCode != 1 || len(res.Diagnostics) == 0 {
		t.Fatalf("want diagnostics, got %+v", res)
	}
	if res.Diagnostics[0].Line == 0 {
		t.Fatalf("diagnostic line not parsed: %+v", res.Diagnostics[0])
	}
}

func TestRunTimeoutKillsGroup(t *testing.T) {
	loops := `package main

import "time"

func main() { for { time.Sleep(time.Second) } }
`
	start := time.Now()
	res, err := Run(context.Background(), Request{
		Files:    map[string]string{"main.go": loops},
		TimeoutS: 2,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !res.TimedOut || res.ExitCode != -1 {
		t.Fatalf("want timeout result, got %+v", res)
	}
	if elapsed := time.Since(start); elapsed > 6*time.Second {
		t.Fatalf("timeout kill took too long: %v", elapsed)
	}
}

// TestRunTimeoutKillsChildren checks the process-TREE kill, not just the direct
// child: the program starts a child that would write a marker file after 3s,
// and the run times out after 1s. Unix kills the process group, Windows the
// Job Object (proc_*.go); if either missed the child, the marker appears.
func TestRunTimeoutKillsChildren(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "child-survived")
	prog := fmt.Sprintf(`package main

import (
	"os"
	"os/exec"
	"time"
)

func main() {
	if len(os.Args) > 1 {
		time.Sleep(3 * time.Second)
		_ = os.WriteFile(%q, []byte("alive"), 0o600)
		return
	}
	child := exec.Command(os.Args[0], "child")
	child.Stdout = os.Stdout // holds the output pipe open, like a real orphan
	if err := child.Start(); err != nil {
		panic(err)
	}
	for {
		time.Sleep(time.Second)
	}
}
`, marker)
	res, err := Run(context.Background(), Request{
		Files:    map[string]string{"main.go": prog},
		TimeoutS: 1,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !res.TimedOut {
		t.Fatalf("want timeout result, got %+v", res)
	}
	// A surviving child would hold stdout open until pipeGrace expires.
	if res.DurationMs >= (time.Second + pipeGrace).Milliseconds() {
		t.Errorf("run took %dms: output pipe stayed open, child likely survived", res.DurationMs)
	}
	time.Sleep(4 * time.Second) // past the child's 3s marker write
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("child process outlived the run's timeout")
	}
}

// TestHelperProcess is not a test. The tree tests below run this test binary
// as the lesson program (the pattern os/exec's own tests use), so they need
// no compiler and run unchanged on Windows and Linux. GORUN_HELPER picks the
// role; GORUN_MARKER is where a surviving child leaves proof of life, and
// GORUN_STARTED, if set, where it says at once that it runs.
func TestHelperProcess(t *testing.T) {
	switch os.Getenv("GORUN_HELPER") {
	case "child":
		if started := os.Getenv("GORUN_STARTED"); started != "" {
			_ = os.WriteFile(started, nil, 0o600)
		}
		time.Sleep(1500 * time.Millisecond)
		_ = os.WriteFile(os.Getenv("GORUN_MARKER"), []byte("alive"), 0o600)
	case "backend":
		// Stands in for the backend: runs a child as its lesson program and
		// is killed by TestProcessTreeDiesWithBackend while the child runs.
		self, _ := os.Executable()
		env := append(os.Environ(), "GORUN_HELPER=child")
		runProcess(context.Background(), filepath.Dir(os.Getenv("GORUN_MARKER")), time.Minute, "", env, self, "-test.run=^TestHelperProcess$")
	case "parent-exits", "parent-hangs":
		self, _ := os.Executable() // not os.Args[0]: it may be relative to another directory
		child := exec.Command(self, "-test.run=^TestHelperProcess$")
		child.Env = append(os.Environ(), "GORUN_HELPER=child")
		child.Stdout = os.Stdout // holds the output pipe open, like a real orphan
		if err := child.Start(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		fmt.Println("child started")
		if os.Getenv("GORUN_HELPER") == "parent-hangs" {
			time.Sleep(time.Minute)
		}
	default:
		return // the real test run
	}
	os.Exit(0)
}

// runHelper runs this test binary as a lesson program in the given role.
func runHelper(ctx context.Context, t *testing.T, role, marker string) Result {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	env := append(os.Environ(), "GORUN_HELPER="+role, "GORUN_MARKER="+marker)
	return runProcess(ctx, t.TempDir(), 10*time.Second, "", env, self, "-test.run=^TestHelperProcess$")
}

// assertChildDied waits past the child's marker write.
func assertChildDied(t *testing.T, marker string) {
	t.Helper()
	time.Sleep(2500 * time.Millisecond)
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the child outlived the run")
	}
}

// A program that exits normally but leaves a child running: the child dies
// when the run ends, and since it held stdout, the run must not wait
// pipeGrace for it either (Vitest workers or Edge left behind look the same).
func TestRunProcessKillsLeftoversAfterExit(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "child-survived")
	res := runHelper(context.Background(), t, "parent-exits", marker)
	if res.ExitCode != 0 || res.TimedOut || res.Stdout != "child started\n" || res.Stderr != "" {
		t.Fatalf("want a clean exit, got %+v", res)
	}
	if res.DurationMs >= pipeGrace.Milliseconds() {
		t.Errorf("run took %dms: the child kept the output pipe open", res.DurationMs)
	}
	assertChildDied(t, marker)
}

// The learner closes the tab: the request context is cancelled, not timed
// out, and the whole tree dies all the same.
func TestRunProcessCancelKillsTree(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "child-survived")
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(500*time.Millisecond, cancel)
	res := runHelper(ctx, t, "parent-hangs", marker)
	if res.TimedOut || res.ExitCode == 0 || res.Stdout != "child started\n" {
		t.Fatalf("want a cancelled run, got %+v", res)
	}
	if res.DurationMs >= (time.Second + pipeGrace).Milliseconds() {
		t.Errorf("run took %dms after a cancel at 500ms", res.DurationMs)
	}
	assertChildDied(t, marker)
}

// waitForFile reports whether path appears within 10 s.
func waitForFile(path string) bool {
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// The backend dies (here: is killed) while a lesson program runs. Nothing in
// the backend is left to kill the tree, so the OS must: on Windows the job's
// kill-on-close flag, when the dead backend's job handle closes; on Linux
// the child's parent-death signal. Without them the child writes its marker.
func TestProcessTreeDiesWithBackend(t *testing.T) {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		t.Skip("only Windows (kill-on-close job) and Linux (Pdeathsig) end the program when the backend dies")
	}
	dir := t.TempDir()
	started, marker := filepath.Join(dir, "started"), filepath.Join(dir, "child-survived")
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	backend := exec.Command(self, "-test.run=^TestHelperProcess$")
	backend.Env = append(os.Environ(), "GORUN_HELPER=backend", "GORUN_STARTED="+started, "GORUN_MARKER="+marker)
	if err := backend.Start(); err != nil {
		t.Fatal(err)
	}
	if !waitForFile(started) {
		_ = backend.Process.Kill()
		t.Fatal("the lesson program never started")
	}
	// The child is in its job as soon as its runtime runs at all; the pause
	// only keeps a slow machine from turning that into a race.
	time.Sleep(200 * time.Millisecond)
	_ = backend.Process.Kill()
	_ = backend.Wait()
	assertChildDied(t, marker)
}

// The build step runs under the same tree handling as the program: when the
// request is cancelled during `go build`, the tool go started dies too, not
// only go. A -toolexec wrapper stands in for the compiler: it is this test
// binary as a child that takes 1.5 s, started by go on its first tool call.
func TestGoToolCancelKillsToolTree(t *testing.T) {
	dir, marks := t.TempDir(), t.TempDir()
	if err := seed(dir, hello(map[string]string{"main.go": helloMain}), goScaffold); err != nil {
		t.Fatal(err)
	}
	started, marker := filepath.Join(marks, "started"), filepath.Join(marks, "child-survived")
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	env := append(os.Environ(), "GOPROXY=off", "GORUN_HELPER=child", "GORUN_STARTED="+started, "GORUN_MARKER="+marker)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		waitForFile(started)
		cancel()
	}()
	toolexec := `-toolexec="` + self + `" -test.run=^TestHelperProcess$`
	res := goTool(ctx, dir, env, "build", toolexec, "-o", filepath.Join(dir, binName), ".")
	if _, err := os.Stat(started); err != nil {
		t.Fatalf("go never started the wrapped tool: %+v", res)
	}
	if res.ExitCode == 0 || res.TimedOut {
		t.Fatalf("want a cancelled build, got %+v", res)
	}
	assertChildDied(t, marker)
}

// timeout_s reaches go test as -timeout: a hanging test ends in Go's own
// panic, which names the test and dumps the goroutines, and go test reports
// FAIL. The outer kill must not come first: it would leave a bare timedOut.
func TestRunTestsTimeoutIsGoTestPanic(t *testing.T) {
	src := `package lesson

import (
	"testing"
	"time"
)

func TestHangs(t *testing.T) { time.Sleep(time.Hour) }
`
	res, err := RunTests(context.Background(), Request{Files: map[string]string{"hang_test.go": src}, TimeoutS: 2})
	if err != nil {
		t.Fatalf("run tests: %v", err)
	}
	for _, want := range []string{"panic: test timed out after 2s", "TestHangs", "goroutine ", "FAIL\tlesson"} {
		if !strings.Contains(res.Stdout, want) {
			t.Errorf("stdout lacks %q", want)
		}
	}
	if res.TimedOut || res.ExitCode != 1 || t.Failed() {
		t.Fatalf("want go test's own timeout failure, got exit %d timedOut %v\nstdout:\n%s\nstderr:\n%s", res.ExitCode, res.TimedOut, res.Stdout, res.Stderr)
	}
}

// Byte-capped output must not end in half a character: run a program that
// prints Turkish past the cap and check the JSON-bound text is valid UTF-8.
func TestRunOutputCapKeepsUTF8(t *testing.T) {
	prog := `package main

import (
	"fmt"
	"strings"
)

func main() { fmt.Print("x" + strings.Repeat("ğ", 40000)) }
`
	res, err := Run(context.Background(), hello(map[string]string{"main.go": prog}))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !res.Truncated || !utf8.ValidString(res.Stdout) || !strings.HasSuffix(res.Stdout, "ğ\n[output truncated]") {
		t.Fatalf("truncated=%v valid=%v tail=%q", res.Truncated, utf8.ValidString(res.Stdout), res.Stdout[len(res.Stdout)-30:])
	}
}

func TestRunOutputCap(t *testing.T) {
	spam := `package main

import "fmt"

func main() { for i := 0; i < 100000; i++ { fmt.Println(i) } }
`
	res, err := Run(context.Background(), hello(map[string]string{"main.go": spam}))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !res.Truncated {
		t.Fatalf("want truncated, got %+v", res)
	}
	if len(res.Stdout) > OutputCap+64 {
		t.Fatalf("stdout over cap: %d", len(res.Stdout))
	}
}

func TestSeedRejectsPathEscape(t *testing.T) {
	for _, name := range []string{"../evil.go", "/abs.go"} {
		_, err := Run(context.Background(), hello(map[string]string{name: "package main"}))
		if err == nil || !strings.Contains(err.Error(), "illegal file name") {
			t.Fatalf("%q: want rejection, got %v", name, err)
		}
	}
}

func TestCappedBufferStopsButDoesNotBlock(t *testing.T) {
	cb := cappedBuffer{cap: 8}
	n, err := cb.Write(bytes.Repeat([]byte("x"), 100))
	if err != nil || n != 100 {
		t.Fatalf("write must report full length, got n=%d err=%v", n, err)
	}
	// cap + the truncation marker the buffer appends.
	if got := cb.String(); !strings.HasSuffix(got, "[output truncated]") {
		t.Fatalf("want truncation marker, got %q", got)
	}
}

func TestRunTestsMode(t *testing.T) {
	main := `package main

import "fmt"

func Add(a, b int) int { return a + b }

func main() { fmt.Println(Add(1, 2)) }
`
	tests := `package main

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("1+2 != 3?")
	}
}
`
	res, err := RunTests(context.Background(), Request{
		Files: map[string]string{"main.go": main, "main_test.go": tests},
	})
	if err != nil {
		t.Fatalf("run tests: %v", err)
	}
	if res.ExitCode != 0 || !strings.Contains(res.Stdout, "--- PASS: TestAdd") {
		t.Fatalf("want passing test output, got exit=%d out=%q", res.ExitCode, res.Stdout)
	}

	failing := strings.Replace(tests, "!= 3", "!= 999", 1)
	res, err = RunTests(context.Background(), Request{
		Files: map[string]string{"main.go": main, "main_test.go": failing},
	})
	if err != nil {
		t.Fatalf("run failing tests: %v", err)
	}
	if res.ExitCode != 1 || !strings.Contains(res.Stdout, "--- FAIL: TestAdd") {
		t.Fatalf("want failing test result, got exit=%d out=%q", res.ExitCode, res.Stdout)
	}
}

// ts-node: annotations are stripped and the program runs as an ES module;
// Node 22's "type stripping is experimental" warning must not reach stderr.
func TestRunNodeStripsTypes(t *testing.T) {
	src := "type Shape = { kind: 'square'; s: number };\n" +
		"const area = (x: Shape): number => x.s ** 2;\n" +
		"const name: string = 'İstanbul';\n" +
		"console.log(area({ kind: 'square', s: 3 }), name.length, typeof import.meta.url);\n"
	res, err := RunNode(context.Background(), hello(map[string]string{"main.ts": src}))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.ExitCode != 0 || res.Stdout != "9 8 string\n" || res.Stderr != "" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if len(res.Files) != 0 {
		t.Fatalf("scaffold package.json reported as written: %+v", res.Files)
	}
}

// Stripping cannot erase an enum (it generates code): Node refuses it. The
// ts.node lessons teach exactly this limit, so it must surface as a result.
func TestRunNodeRejectsNonErasableSyntax(t *testing.T) {
	res, err := RunNode(context.Background(), hello(map[string]string{
		"main.ts": "enum Color { Red }\nconsole.log(Color.Red);\n",
	}))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.ExitCode == 0 || !strings.Contains(res.Stderr, "ERR_UNSUPPORTED_TYPESCRIPT_SYNTAX") {
		t.Fatalf("want a non-erasable syntax error, got %+v", res)
	}
}

func TestRunNodeTimeoutKillsProgram(t *testing.T) {
	res, err := RunNode(context.Background(), Request{
		Files:    map[string]string{"main.ts": "while (true) {}\n"},
		TimeoutS: 1,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !res.TimedOut || res.DurationMs >= (time.Second+pipeGrace).Milliseconds() {
		t.Fatalf("want a timeout within the grace period, got %+v", res)
	}
}

func TestNodeEntry(t *testing.T) {
	tests := []struct {
		files map[string]string
		want  string
	}{
		{map[string]string{"main.ts": "", "lib.ts": ""}, "main.ts"},
		{map[string]string{"cli.ts": "", "data.json": ""}, "cli.ts"},
		{map[string]string{"a.ts": "", "b.ts": ""}, ""},
	}
	for _, tc := range tests {
		got, err := nodeEntry(tc.files)
		if got != tc.want || (tc.want == "") != (err != nil) {
			t.Errorf("nodeEntry(%v) = %q, %v; want %q", tc.files, got, err, tc.want)
		}
	}
}

// golang.org/x/sync and x/text come with the pgx module set, offline.
func TestRunBuildsWithXSyncAndXTextOffline(t *testing.T) {
	src := `package main

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func main() {
	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error { return nil })
	fmt.Println(g.Wait(), cases.Upper(language.Turkish).String("istanbul"))
}
`
	res, err := Run(context.Background(), hello(map[string]string{"main.go": src}))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.ExitCode != 0 || res.Stdout != "<nil> İSTANBUL\n" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

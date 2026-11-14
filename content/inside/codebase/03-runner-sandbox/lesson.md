---
id: inside.codebase.runner-sandbox
title: "The runner sandbox and its threat model"
track: inside
module: codebase
order: 3
level: advanced
minutes: 15
since: go1.25
prerequisites: [go.services.tcp-exec, go.concurrency.context, go.concurrency.patterns]
compare_with: [inside.codebase.sql-rollback, inside.codebase.compose-topology, design.architecture.security, go.files-io.directories]
tags: [sandbox, os-exec, timeouts, process-tree, output-cap, semaphore, threat-model]
summary: >
  ADR-0003 promised timeouts, output caps, a process-tree kill and two run slots;
  internal/gorun and internal/runexec deliver them in both modes. The container
  limits exist only in Docker mode: in local mode (ADR-0008), lesson code runs as
  you, with your files, network and environment.
examples:
  - id: timeout-and-cap
    file: examples/01_timeout_and_cap.go
    title: "A runaway program: the deadline kills it, the cap keeps 40 bytes"
    runtime: go
    check: exact
    annotations:
      - lines: "16-35"
        note: >
          gorun's `cappedBuffer`, with the cap cut from 64 KB to 40 bytes. It is
          an `io.Writer` the output is copied into as it arrives, so the cap is
          enforced while the output streams in, not after the program has
          filled memory.
      - lines: "23-25"
        note: >
          Past the cap it still reports success. Returning an error would stop
          the copying goroutine, the pipe would fill, and the program would
          block on its next write: a hang instead of a truncation.
      - lines: "30-31"
        note: >
          The cut backs up to a character boundary. Byte 40 is the first half
          of the two-byte "ı", so the output ends in "sat", not in half a letter
          that the JSON encoder would turn into U+FFFD on its way to you. The
          check looks at the buffer, not only at p, because a character can
          arrive split across two writes.
      - lines: "37-48"
        note: >
          A UTF-8 character is at most four bytes, so only the last three can
          start one that isn't finished. `utf8.FullRune` counts invalid bytes
          as whole, so binary output is cut where it is.
      - lines: "61-67"
        note: >
          The real `runProcess` in miniature: a context deadline (5 s by
          default, at most 15), output caps per stream, and `WaitDelay`
          (gorun's `pipeGrace`, 2 s) bounding how long Wait keeps reading after
          the process is gone. The real one also reads the pipes itself and
          kills the whole process tree whenever the run ends (example 2).
      - lines: "69-76"
        note: >
          Check the context before the exit code. A killed process reports
          exit status 1 on Windows and "signal: killed" (-1) on Linux; gorun
          normalizes every timeout to `timedOut: true, exitCode: -1` so the UI
          and the judge see the same thing on both.
  - id: orphan-grandchild
    file: examples/02_orphan_grandchild.go
    title: "Why the whole process tree must die, and why WaitDelay exists"
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "16-22"
        note: >
          The lesson program starts a process of its own that inherits its
          stdout, then hangs. This is the shape of gorun's own
          `TestRunTimeoutKillsChildren`.
      - lines: "31"
        note: >
          `exec.CommandContext`'s default Cancel kills only the direct child.
          gorun replaces it in `proc_unix.go` (own process group, SIGKILL to
          the negative PID) and `proc_windows.go` (a Job Object per run,
          ended with TerminateJobObject), which a single cross-platform file
          can't show.
      - lines: "34-36"
        note: >
          With `WaitDelay` 0, Run reads the pipe until EOF, and EOF only comes
          when every holder of the write end has exited, including the
          orphaned grandchild 3 s later. With 300 ms, Run closes the pipe and
          returns. The os/exec docs describe exactly this case.
      - lines: "42-45"
        note: >
          A tree kill done by hand: the grandchild told us its PID, so we can
          kill it. Without that it would have run to the end on its own. gorun
          doesn't need the PID: after Wait returns it kills whatever is left in
          the program's process group or Job Object, so a lesson program that
          exits normally can't leave a background process behind either.
  - id: two-slots
    file: examples/03_two_slots.go
    title: "At most two runs at once: a channel as a semaphore"
    runtime: go
    check: exact
    annotations:
      - lines: "11-13"
        note: >
          A buffered channel of empty structs is the semaphore: sending takes a
          slot, receiving frees it, `cap` is the limit. runexec uses exactly
          this, with `maxConcurrent = 2`.
      - lines: "16-23"
        note: >
          Three ways out of the wait: a slot, the request's context (the
          browser went away), or `queueWait` (10 s in runexec). The last two
          become 503 "server busy". `time.After` in a one-shot select is fine;
          since Go 1.23 an unfired timer no longer leaks.
      - lines: "18"
        note: >
          The slot is released by defer, so a panicking or failing run can't
          keep it forever.
      - lines: "35-37"
        note: >
          Waiting on `len(e.slots)` makes the demo deterministic: runs 1 and 2
          hold both slots before run 3 asks. Without it, the scheduler could
          let run 3 in first.
use_cases:
  - "Running code you didn't write, or wrote carelessly, without letting it hang or flood the process that started it"
  - "Any os/exec call in a server: a deadline, capped output, a bounded Wait, and a tree kill"
  - "Bounding concurrent expensive work (builds, conversions, renders) with a channel semaphore and a 503 when full"
strengths:
  - "Every limit is a constant at the top of `gorun.go` or `runexec.go`, readable in one screen, and exercised by `gorun_test.go` (timeout, tree kill, output cap)"
  - "One package, `runexec`, serves both modes: the Docker runner and the in-process backend can't disagree about timeouts or caps"
  - "Compile errors, panics and timeouts come back as data (`exitCode`, `timedOut`, `truncated`, `diagnostics`), never as HTTP errors"
  - "Unlike a Java `Process.destroy()`, which ends only the direct child, every run's end kills descendants on both operating systems: on timeout, on a cancelled request, and after a normal exit. The `go build` and `go vet` steps get the same treatment"
  - "On Windows even a crashed backend takes the lesson's processes with it: the run's Job Object is kill-on-close, and the backend holds its only handle"
weaknesses:
  - "Local mode has no isolation at all: lesson code runs as the owner, can read and delete any of the owner's files, reach the network and the `stacklab` database on 127.0.0.1:5432, and inherits the backend's environment variables"
  - "A process that leaves the tree escapes the kill: `setsid` on Unix; on Windows one started through WMI, or in the instant between the program's start and its assignment to the Job Object. runProcess then stops reading its output after 2 s"
  - "If the backend itself dies mid-run, Linux kills only the lesson's direct child (its parent-death signal, `proc_linux.go`): what that program started keeps running. macOS kills nothing"
  - "Docker mode's real limits (memory, pids, CPU, read-only root, no internet) live in `docker-compose.yml`; nothing in Go checks they are there"
pitfalls:
  - title: "Trusting exec.CommandContext to stop a program"
    detail: "Its default Cancel kills one process. Children survive and can hold the output pipe open, so Wait blocks until they exit. Set `cmd.Cancel` to kill the tree and `cmd.WaitDelay` to bound Wait, and kill what is left after Wait returns too: Cancel only fires while the program runs."
    bad: "cmd := exec.CommandContext(ctx, bin)\ncmd.Stdout = &buf\nerr := cmd.Run()"
    good: "cmd := exec.CommandContext(ctx, bin)\ntree := newProcessTree(cmd) // sets cmd.Cancel; proc_unix.go / proc_windows.go\ncmd.WaitDelay = pipeGrace\ncmd.Stdout = stdoutPipe\nerr := cmd.Start()\ntree.started()\nerr = cmd.Wait()\ntree.killLeftovers()"
  - title: "An io.Writer that errors when full"
    detail: "exec stops copying, the pipe fills, and the child blocks on write until the timeout. Swallow the excess and record that you did."
  - title: "Reading ExitCode for a timeout"
    detail: "It differs by OS (1 on Windows, -1 on Unix for a kill). Check `ctx.Err()` first and report the timeout as such."
  - title: "Reading ADR-0003 as a description of local mode"
    detail: "Its container limits apply to Docker mode only (ADR-0008). Locally, the per-run hygiene in gorun and runexec is all there is."
quiz:
  - q: "A lesson program prints in an endless loop. What stops it?"
    options:
      - "The 64 KB output cap"
      - "The run deadline (5 s by default): the context fires and the process tree is killed"
      - "The 2-slot semaphore"
      - "Docker's memory limit, in both modes"
    answer: 1
    why: "The cap only stops storing output; it keeps accepting writes so the program never blocks. The deadline kills it."
  - q: "With `WaitDelay` 0, why can Run block long after the killed child has exited?"
    options:
      - "A grandchild that inherited stdout still holds the pipe, so EOF never arrives until it exits"
      - "Run always waits for the full timeout"
      - "The kernel delays process exit"
    answer: 0
    why: "Run waits for the copying goroutine, which ends at EOF. WaitDelay bounds that wait. gorun removes the cause instead: it reads the pipes itself and kills the tree as soon as the program has exited, so EOF follows at once."
  - q: "Which of these protections exists in local mode (npm start)?"
    options:
      - "The 512 MB memory limit"
      - "No internet access"
      - "The run timeout and the output cap"
      - "A read-only root filesystem"
    answer: 2
    why: "Memory, network and filesystem limits come from docker-compose.yml. Timeouts, caps, the tree kill and the two slots are Go code in gorun and runexec, so they apply in both modes."
further_reading:
  - title: "package os/exec (Cmd.Cancel, Cmd.WaitDelay)"
    url: "https://pkg.go.dev/os/exec#Cmd"
  - title: "package context"
    url: "https://pkg.go.dev/context"
  - title: "path/filepath: IsLocal"
    url: "https://pkg.go.dev/path/filepath#IsLocal"
  - title: "encoding/json: invalid UTF-8 in strings"
    url: "https://pkg.go.dev/encoding/json#Marshal"
---

## Concept

ADR-0003 states the threat model plainly: the sandbox protects the owner from their own
mistakes (infinite loops, fork bombs, huge output, deleted files, runaway memory), not
from a determined attacker. ADR-0008 then moved the default run mode out of the container
and into the backend process. So the useful question is: which promise lives in which
place, and does it still hold when you run `npm start`?

In Go code, so in **both modes**:

- **Fresh temp dir per run**: `newWorkDir` uses `os.MkdirTemp`, removed by
  `defer os.RemoveAll`. File names must pass `filepath.IsLocal`, which on Windows also
  rejects `C:x`, `\x` and device names like `NUL`. ts-test runs under `.lesson-runs/` in
  the project instead, so Node finds the app's `node_modules`.
- **Timeouts**: `compileTimeout` 20 s; run `defaultRunTTL` 5 s, at most `maxRunTTL` 15 s
  (`runapi.Validate` rejects more); go-test and ts-test take `timeout_s` the same way,
  but 15 s when it is unset, because starting Vitest or Edge, or running a table of
  cases, is part of the run. go-test hands it to `go test -timeout`, so a hanging test
  ends in Go's own "test timed out" panic with a goroutine dump; the 20 s build
  allowance comes on top, and the outer kill only 3 s after that. The whole request:
  `runBudget` 40 s in runexec.
- **Output caps**: `OutputCap` 64 KB per stream with `[output truncated]`, cut back to a
  whole character so Turkish text doesn't end in half a letter, and `truncated` in the
  result.
- **Tree kill whenever a run ends**: on timeout, on a cancelled request, and after a
  normal exit that left processes behind (`processTree` in `proc_unix.go` and
  `proc_windows.go`), for the `go build` and `go vet` steps too. If the backend itself
  dies, Windows still ends the whole job; Linux only the direct child (`proc_linux.go`).
- **Two concurrent runs**: runexec's `slots` channel, 10 s queue wait, then 503.
- **Offline builds**: `GOPROXY=off`, so `go build` can't download anything.

In `docker-compose.yml`, so in **Docker mode only**: `user: "65532:65532"`,
`read_only: true` with tmpfs for `/tmp` and `/work`, `mem_limit: 512m`, `cpus: 1`,
`pids_limit: 128`, and networks marked `internal: true`.

## Coming from C/C++/Java/Python/JS

- **C/POSIX**: `setpgid` plus `kill(-pgid, SIGKILL)` is the classic way to kill a tree;
  `proc_unix.go` does exactly that through `SysProcAttr{Setpgid: true}`. Windows has no
  process groups to signal; its equivalent is a Job Object, which every descendant
  joins automatically.
- **Java**: `Process.destroy()` ends one process; you need `ProcessHandle.descendants()`
  to find the rest. Go's `exec.CommandContext` has the same one-process default.
- **Python**: `subprocess.run(timeout=...)` kills the direct child only; its children
  survive, as with Go's default Cancel. `start_new_session=True` plus `os.killpg` is the
  POSIX fix, the same idea as `Setpgid`.
- **Node**: `child_process.spawn` with `detached: true` and `process.kill(-pid)` is the
  Unix pattern; `scripts/verify-lessons.mjs` does that for the headless browser, and
  `taskkill /T /F` on Windows.

## How it works

`gorun.runProcess` is the one function every server runtime goes through: Go programs
and their `go build` (and `go vet`) step, `go test`, Node for ts-node and Vitest or
Playwright for ts-test:

```go
runCtx, cancel := context.WithTimeout(ctx, ttl)
cmd := exec.CommandContext(runCtx, name, args...)
tree := newProcessTree(cmd) // Setpgid (Unix); cmd.Cancel kills the tree
cmd.WaitDelay = pipeGrace   // 2 s
cmd.Stdout = stdoutW        // an os.Pipe; gorun copies the read end into a 64 KB cappedBuffer
runErr := cmd.Start()
tree.started()              // Windows: put the program in a new Job Object
runErr = cmd.Wait()         // returns when the program exits
tree.killLeftovers()        // whatever it left running
```

The context comes from the HTTP request, so closing the browser tab cancels the run too.
The result records `TimedOut` from `runCtx.Err()` and normalizes the exit code to -1.

gorun reads the pipes itself rather than handing exec an `io.Writer`. With a Writer,
Wait keeps copying until every process holding the pipe has exited, so a background
child left by the program would hold up the run until `WaitDelay` gave up, and only then
could it be killed. With the pipe's write end passed as an `*os.File`, Wait returns as
soon as the program exits, the leftovers are killed right then, and the copies reach EOF.

On Windows the kill is `TerminateJobObject`: the program joins a fresh Job Object right
after it starts, and every process it starts later joins it too. `taskkill /T` finds
descendants through their parent, so it can't reach the children of a program that has
already exited; it remains only as the fallback when no job can be created. The job is
also kill-on-close (`JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`, set with
`SetInformationJobObject`), and the backend holds its only handle until the run is over.
If the backend crashes, Windows closes that handle and the job's processes end with it.
The four kernel32 functions come from the standard library's `syscall.NewLazyDLL`, so the
dependency ADR-0008 wanted to avoid (`golang.org/x/sys`) is still not needed; the one
structure `SetInformationJobObject` takes is declared in `proc_windows.go`, field by
field from Microsoft's documentation.

On Linux, `SysProcAttr.Pdeathsig` asks the kernel to SIGKILL the program when the
backend dies. It reaches the direct child only. And the "parent" is the thread that
forked, not the process, so the Go docs warn that the signal can come early, when that
thread exits (go.dev/issue/27505). The runtime ends a thread when a goroutine locked to
it with `runtime.LockOSThread` exits; nothing that calls gorun locks threads.

## Design notes

Read against the threat model, the code is honest about local mode: ADR-0008 lists what
is lost and accepts it for a single-user app bound to 127.0.0.1. What that means in
practice deserves to be said plainly:

- **A local lesson program is just a program you ran.** It runs as you, sees your files
  (`~/.ssh`, the repo), can reach the internet, and inherits the backend's environment
  (`cmd.Env` is nil, so `os.Environ()`), including any `STACKLAB_*` variables you set. It
  can also connect to the app database with the default credentials in `config.go`.
- **Docker mode is less isolated than its comments say.** `cmd/runner/main.go` says "The
  backend is its only network peer", but in `docker-compose.yml` the runner sits on
  `backend-net` together with `app-db` and `mocksvc`, and on `runner-net` with
  `playground-db`. And the playground role there is created by the postgres image from
  `POSTGRES_USER`, which the image documents as a superuser, so the "non-superuser role"
  of ADR-0003 holds only in local mode (`scripts/setup-local-db.sql`).
- **A gap left on Linux:** a backend crash kills the lesson program but not what it
  started. A PID namespace per run would close it, since the kernel ends a namespace's
  processes when its first process dies; Docker mode gets that from the container.
- **If the app ever leaves 127.0.0.1, none of this is a sandbox.** ADR-0003 already says
  so: rebuild the runner on gVisor, Firecracker or a separate host.

//go:build windows

// proc_windows.go: process-tree handling on Windows (local mode, ADR-0008).
//
// Each run gets a Job Object. The program is assigned to it right after it
// starts; every process it starts from then on joins the job by itself, and
// TerminateJobObject ends them all at once: on timeout or a cancelled request
// while the program runs, and after it has exited and left children behind
// (a lesson's background process, Vitest workers, Edge under Playwright).
// Windows has no process groups to signal, and `taskkill /T` finds
// descendants through their parent, so it can't reach the children of a
// program that has already exited.
//
// The standard library's syscall package doesn't wrap the job functions, so
// the three kernel32 calls are loaded with syscall.NewLazyDLL. ADR-0008 had
// rejected Job Objects only because golang.org/x/sys/windows is outside the
// dependency allow-list; this needs nothing outside the standard library.
// kernel32.dll is a KnownDLL, always loaded from System32, so loading it by
// name can't pick up a planted copy.
//
// The job sets no breakaway limit, so no child can leave it, even when a job
// the backend itself runs in would allow breakaway: the immediate job decides
// (Microsoft's "Nested Jobs" page; nesting needs Windows 8). Escapes left: a
// child started between CreateProcess and the assignment, a window far
// shorter than the runtime start-up a Go or Node program goes through before
// user code runs; and processes created through WMI
// (Win32_Process.Create), which never join the creator's job. If the job
// can't be set up (a job with UI limits can't be nested), the timeout path
// falls back to `taskkill /T /F`, which works while the program is alive, and
// leftovers after a normal exit are not killed.

package gorun

import (
	"os/exec"
	"strconv"
	"sync"
	"syscall"
)

// exeSuffix is appended to the compiled lesson binary's name: Windows only
// executes files with an executable extension.
const exeSuffix = ".exe"

// processSetQuota is PROCESS_SET_QUOTA, which AssignProcessToJobObject needs
// on the process handle; package syscall has no constant for it.
const processSetQuota = 0x0100

var (
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procCreateJobObjectW         = kernel32.NewProc("CreateJobObjectW")
	procAssignProcessToJobObject = kernel32.NewProc("AssignProcessToJobObject")
	procTerminateJobObject       = kernel32.NewProc("TerminateJobObject")
)

// processTree is a lesson program and every process it starts.
type processTree struct {
	cmd *exec.Cmd
	// mu: exec may call cmd.Cancel (kill) from its own goroutine while
	// started is still putting the program into the job.
	mu  sync.Mutex
	job syscall.Handle // 0 until the program is in a job of its own
}

// newProcessTree prepares cmd, before Start, and makes the context deadline
// (cmd.Cancel) kill the whole tree.
func newProcessTree(cmd *exec.Cmd) *processTree {
	t := &processTree{cmd: cmd}
	cmd.Cancel = t.kill
	return t
}

// started puts the program, right after cmd.Start, into a new job. On any
// failure the job stays 0 and kill falls back to taskkill.
func (t *processTree) started() {
	t.mu.Lock()
	defer t.mu.Unlock()
	job, _, _ := procCreateJobObjectW.Call(0, 0)
	if job == 0 {
		return
	}
	proc, err := syscall.OpenProcess(processSetQuota|syscall.PROCESS_TERMINATE, false, uint32(t.cmd.Process.Pid))
	if err != nil {
		_ = syscall.CloseHandle(syscall.Handle(job))
		return
	}
	defer syscall.CloseHandle(proc)
	if ok, _, _ := procAssignProcessToJobObject.Call(job, uintptr(proc)); ok == 0 {
		_ = syscall.CloseHandle(syscall.Handle(job))
		return
	}
	t.job = syscall.Handle(job)
}

// kill ends the program and everything it started, with exit code 1 (what
// taskkill /F gives too). It is cmd.Cancel, so the program is still running.
func (t *processTree) kill() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.job != 0 {
		if ok, _, err := procTerminateJobObject.Call(uintptr(t.job), 1); ok == 0 {
			return err
		}
		return nil
	}
	pid := strconv.Itoa(t.cmd.Process.Pid)
	if err := exec.Command("taskkill", "/T", "/F", "/PID", pid).Run(); err != nil {
		return t.cmd.Process.Kill()
	}
	return nil
}

// killLeftovers ends whatever the program left running when it exited.
// Without a job there is no way to find those processes: taskkill would need
// the exited program to walk the tree from.
func (t *processTree) killLeftovers() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.job != 0 {
		_, _, _ = procTerminateJobObject.Call(uintptr(t.job), 1)
	}
}

// release closes the job handle. Processes still in the job would keep
// running, which is why runProcess calls killLeftovers first.
func (t *processTree) release() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.job != 0 {
		_ = syscall.CloseHandle(t.job)
		t.job = 0
	}
}

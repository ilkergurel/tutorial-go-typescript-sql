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
// The job is also set to kill on close (JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE):
// when its last handle closes, every process still in it ends. The backend
// holds the only handle, from the program's start until the run is over
// (CreateJobObjectW without security attributes makes it non-inheritable, so
// no child holds a copy), and Windows closes it when the backend dies, so a
// crashed or killed backend no longer leaves lesson processes running. If
// the flag can't be set, the job still kills the tree while the backend lives.
//
// The standard library's syscall package doesn't wrap the job functions, so
// the four kernel32 calls are loaded with syscall.NewLazyDLL. ADR-0008 had
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
	"unsafe"
)

// exeSuffix is appended to the compiled lesson binary's name: Windows only
// executes files with an executable extension.
const exeSuffix = ".exe"

// processSetQuota is PROCESS_SET_QUOTA, which AssignProcessToJobObject needs
// on the process handle; package syscall has no constant for it.
const processSetQuota = 0x0100

const (
	// jobObjectExtendedLimitInformation is the JOBOBJECTINFOCLASS value that
	// tells SetInformationJobObject it gets a jobLimits.
	jobObjectExtendedLimitInformation = 9
	// jobObjectLimitKillOnJobClose is JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE, a
	// LimitFlags bit that only the extended structure may carry.
	jobObjectLimitKillOnJobClose = 0x00002000
	// jobLimitsSize is sizeof(JOBOBJECT_EXTENDED_LIMIT_INFORMATION) on 64-bit
	// Windows.
	jobLimitsSize = 144
)

// jobLimits is JOBOBJECT_EXTENDED_LIMIT_INFORMATION, with its first member,
// JOBOBJECT_BASIC_LIMIT_INFORMATION, flattened in and IO_COUNTERS as six
// uint64s, from the winnt.h definitions on Microsoft Learn ("JOBOBJECT_
// EXTENDED_LIMIT_INFORMATION structure", "JOBOBJECT_BASIC_LIMIT_INFORMATION
// structure", "IO_COUNTERS structure"; the class value and the flag are from
// "SetInformationJobObject function" and the basic structure's LimitFlags
// table). LARGE_INTEGER is int64, DWORD uint32, SIZE_T and ULONG_PTR uintptr.
// On amd64 and arm64 Go aligns every field as the C compiler does, padding
// included, so the layout matches: 144 bytes, LimitFlags at offset 16, the
// I/O counters at 64 (TestJobLimitsLayout). On 386 Go aligns int64 to 4
// bytes where C uses 8, so the size differs and setKillOnClose doesn't call.
type jobLimits struct {
	perProcessUserTimeLimit int64
	perJobUserTimeLimit     int64
	limitFlags              uint32
	minimumWorkingSetSize   uintptr
	maximumWorkingSetSize   uintptr
	activeProcessLimit      uint32
	affinity                uintptr
	priorityClass           uint32
	schedulingClass         uint32
	ioCounters              [6]uint64
	processMemoryLimit      uintptr
	jobMemoryLimit          uintptr
	peakProcessMemoryUsed   uintptr
	peakJobMemoryUsed       uintptr
}

var (
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procCreateJobObjectW         = kernel32.NewProc("CreateJobObjectW")
	procSetInformationJobObject  = kernel32.NewProc("SetInformationJobObject")
	procAssignProcessToJobObject = kernel32.NewProc("AssignProcessToJobObject")
	procTerminateJobObject       = kernel32.NewProc("TerminateJobObject")
)

// setKillOnClose makes closing the job's last handle end every process in
// it. A new job has no limits, so a zero structure plus the one flag changes
// nothing else.
func setKillOnClose(job syscall.Handle) error {
	var limits jobLimits
	if unsafe.Sizeof(limits) != jobLimitsSize {
		return syscall.EWINDOWS
	}
	limits.limitFlags = jobObjectLimitKillOnJobClose
	ok, _, err := procSetInformationJobObject.Call(uintptr(job), jobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&limits)), unsafe.Sizeof(limits))
	if ok == 0 {
		return err
	}
	return nil
}

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
	// Before the program joins, so it is never in the job without the flag.
	// An error costs only the clean-up after a backend crash (file comment).
	_ = setKillOnClose(syscall.Handle(job))
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

// release closes the job handle at the end of the run. With kill on close
// set, that alone would end what is still in the job, but runProcess has
// killed it already (killLeftovers), which also covers a job without the
// flag. The flag is for the case where release never runs: a dead backend.
func (t *processTree) release() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.job != 0 {
		_ = syscall.CloseHandle(t.job)
		t.job = 0
	}
}

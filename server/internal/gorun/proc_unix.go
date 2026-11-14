//go:build !windows

// proc_unix.go: process-tree handling on Linux/macOS (Docker mode, WSL).
//
// The program runs in its own process group (Setpgid), so one SIGKILL to the
// negative pid takes down every process it forked, not just the direct child.
// The group outlives its leader: once the program has exited and been reaped,
// the children it left behind are still members, so the same signal ends them
// after a normal exit too. POSIX keeps a pid from being reused while a process
// group with that id has members, so the signal can't reach a stranger; when
// the group is empty it fails with ESRCH, which is ignored. A process that
// leaves the group on purpose (setsid, as daemons do) escapes; runProcess then
// stops reading its output after pipeGrace.
//
// All of that needs a living backend. If the backend dies mid-run, Linux
// still kills the direct child (setParentDeathSignal, proc_linux.go); other
// Unixes don't (proc_otherunix.go).

package gorun

import (
	"os/exec"
	"syscall"
)

// exeSuffix is appended to the compiled lesson binary's name.
const exeSuffix = ""

// processTree is a lesson program and every process it starts.
type processTree struct {
	cmd *exec.Cmd
}

// newProcessTree prepares cmd, before Start, so its tree can be killed, and
// makes the context deadline (cmd.Cancel) kill the whole tree.
func newProcessTree(cmd *exec.Cmd) *processTree {
	t := &processTree{cmd: cmd}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	setParentDeathSignal(cmd.SysProcAttr)
	cmd.Cancel = t.kill
	return t
}

// started runs right after cmd.Start. Nothing to do: the group exists from
// the fork on.
func (t *processTree) started() {}

// kill sends SIGKILL to the program's process group.
func (t *processTree) kill() error {
	return syscall.Kill(-t.cmd.Process.Pid, syscall.SIGKILL)
}

// killLeftovers ends whatever the program left running when it exited.
func (t *processTree) killLeftovers() { _ = t.kill() }

// release frees what the tree holds; a process group needs nothing freed.
func (t *processTree) release() {}

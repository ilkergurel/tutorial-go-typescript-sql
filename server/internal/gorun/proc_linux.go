// proc_linux.go: on Linux the program also dies with the backend.
//
// Pdeathsig asks the kernel to SIGKILL the program when the backend dies, so
// a crash or a kill -9 of the backend doesn't leave a lesson program, a
// go build or a node running (prctl PR_SET_PDEATHSIG, set by os/exec in the
// child between fork and exec). What it doesn't cover:
//   - Only the direct child gets the signal. Processes the program started
//     are re-parented and keep running; their process group outlives the
//     backend, but nobody is left to signal it.
//   - The "parent" is the thread that forked, not the process: the signal
//     also fires if that thread exits first (go.dev/issue/27505). The Go
//     runtime ends a thread when a goroutine locked to it
//     (runtime.LockOSThread) exits without unlocking; nothing that calls
//     gorun locks threads.
//   - In Docker mode it adds little: the runner is the container's PID 1,
//     and when it exits the kernel kills everything in the container.
//
// Linux only: other Unixes keep proc_otherunix.go's no-op. macOS has no such
// setting; FreeBSD has one, but no StackLab mode runs there.

package gorun

import "syscall"

func setParentDeathSignal(attr *syscall.SysProcAttr) { attr.Pdeathsig = syscall.SIGKILL }

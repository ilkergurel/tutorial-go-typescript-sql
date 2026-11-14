//go:build !windows && !linux

// proc_otherunix.go: Unix systems other than Linux get no parent-death
// signal (proc_linux.go), so a backend crash leaves the program running
// there; the process group still ends it whenever a run ends.

package gorun

import "syscall"

func setParentDeathSignal(*syscall.SysProcAttr) {}

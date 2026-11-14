// proc_windows_test.go: the Job Object limit structure gorun hands to
// kernel32. Windows only (by its name) but not behind the integration tag:
// it calls the kernel and starts no process. That the flag does its job (a
// dead backend's lesson processes end) is TestProcessTreeDiesWithBackend, an
// integration test.

package gorun

import (
	"syscall"
	"testing"
	"unsafe"
)

// jobLimits must have the offsets C gives JOBOBJECT_EXTENDED_LIMIT_INFORMATION
// on 64-bit Windows: natural alignment, so a 4-byte DWORD followed by an
// 8-byte SIZE_T or ULONG_PTR leaves 4 bytes of padding.
func TestJobLimitsLayout(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("the layout is defined for 64-bit Windows (amd64, arm64)")
	}
	var l jobLimits
	for _, f := range []struct {
		name      string
		got, want uintptr
	}{
		{"size", unsafe.Sizeof(l), jobLimitsSize},
		{"PerJobUserTimeLimit", unsafe.Offsetof(l.perJobUserTimeLimit), 8},
		{"LimitFlags", unsafe.Offsetof(l.limitFlags), 16},
		{"MinimumWorkingSetSize", unsafe.Offsetof(l.minimumWorkingSetSize), 24},
		{"ActiveProcessLimit", unsafe.Offsetof(l.activeProcessLimit), 40},
		{"Affinity", unsafe.Offsetof(l.affinity), 48},
		{"SchedulingClass", unsafe.Offsetof(l.schedulingClass), 60},
		{"IoInfo", unsafe.Offsetof(l.ioCounters), 64},
		{"ProcessMemoryLimit", unsafe.Offsetof(l.processMemoryLimit), 112},
		{"PeakJobMemoryUsed", unsafe.Offsetof(l.peakJobMemoryUsed), 136},
	} {
		if f.got != f.want {
			t.Errorf("%s at %d, want %d", f.name, f.got, f.want)
		}
	}
}

// The kernel accepts the structure (it rejects a wrong length) and reads the
// flag where gorun put it: QueryInformationJobObject returns it set.
func TestSetKillOnClose(t *testing.T) {
	h, _, err := procCreateJobObjectW.Call(0, 0)
	if h == 0 {
		t.Fatalf("CreateJobObjectW: %v", err)
	}
	job := syscall.Handle(h)
	defer syscall.CloseHandle(job)
	if err := setKillOnClose(job); err != nil {
		t.Fatalf("setKillOnClose: %v", err)
	}
	var got jobLimits
	query := kernel32.NewProc("QueryInformationJobObject")
	if ok, _, err := query.Call(uintptr(job), jobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&got)), unsafe.Sizeof(got), 0); ok == 0 {
		t.Fatalf("QueryInformationJobObject: %v", err)
	}
	if got.limitFlags != jobObjectLimitKillOnJobClose {
		t.Fatalf("LimitFlags = %#x, want only KILL_ON_JOB_CLOSE (%#x)", got.limitFlags, jobObjectLimitKillOnJobClose)
	}
}

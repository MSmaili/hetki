package tmux

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	reportKernProc       = 14
	reportKernProcPID    = 1
	reportProcessExiting = 0x00002000
)

func readReportProcess(ctx context.Context, pid int) (reportProcess, error) {
	if err := ctx.Err(); err != nil {
		return reportProcess{}, err
	}
	if pid <= 0 || pid > math.MaxInt32 {
		return reportProcess{}, errors.New("invalid reporting process PID")
	}
	var proc unix.KinfoProc
	mib := []int32{unix.CTL_KERN, reportKernProc, reportKernProcPID, int32(pid)}
	size := unsafe.Sizeof(proc)
	_, _, errno := syscall.Syscall6(unix.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib[0])), uintptr(len(mib)),
		uintptr(unsafe.Pointer(&proc)), uintptr(unsafe.Pointer(&size)), 0, 0)
	runtime.KeepAlive(mib)
	runtime.KeepAlive(&proc)
	if errno != 0 {
		return reportProcess{}, fmt.Errorf("read reporting process metadata: %w", errno)
	}
	if size != unsafe.Sizeof(proc) || proc.Proc.P_pid != int32(pid) || proc.Proc.P_flag&reportProcessExiting != 0 ||
		proc.Proc.P_starttime.Sec <= 0 || proc.Proc.P_starttime.Usec < 0 || proc.Proc.P_starttime.Usec >= 1_000_000 {
		return reportProcess{}, errors.New("reporting process is unavailable or exiting")
	}
	if err := ctx.Err(); err != nil {
		return reportProcess{}, err
	}
	return reportProcess{pid: pid, parent: int(proc.Eproc.Ppid), birth: fmt.Sprintf("%d:%d", proc.Proc.P_starttime.Sec, proc.Proc.P_starttime.Usec)}, nil
}

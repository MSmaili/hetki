package foreground

import (
	"context"
	"os"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestReadDarwinProcessUsesFixedCapacity(t *testing.T) {
	proc, ok := readDarwinProcess(os.Getpid())
	require.True(t, ok)
	require.EqualValues(t, os.Getpid(), proc.Proc.P_pid)
	_, ok = readDarwinProcess(1_000_000_000)
	require.False(t, ok)
}

func TestDarwinProcessRecheckDetectsIdentityAndJobChanges(t *testing.T) {
	before, ok := readDarwinProcess(os.Getpid())
	require.True(t, ok)
	require.True(t, sameDarwinProcess(before, before))
	for _, mutate := range []func(*unix.KinfoProc){
		func(p *unix.KinfoProc) { p.Proc.P_pid++ },
		func(p *unix.KinfoProc) { p.Proc.P_starttime.Usec++ },
		func(p *unix.KinfoProc) { p.Proc.P_comm[0] ^= 1 },
		func(p *unix.KinfoProc) { p.Proc.P_flag ^= processLP64 },
		func(p *unix.KinfoProc) { p.Eproc.Pgid++ },
		func(p *unix.KinfoProc) { p.Eproc.Tpgid++ },
		func(p *unix.KinfoProc) { p.Eproc.Tdev++ },
	} {
		after := before
		mutate(&after)
		require.False(t, sameDarwinProcess(before, after))
	}
}

func TestDarwinArgumentSizePreflightAndCancellation(t *testing.T) {
	mib := []int32{unix.CTL_KERN, kernProcArgs2, int32(os.Getpid())}
	size, err := boundedSysctl(mib, nil, 0)
	require.NoError(t, err)
	require.Greater(t, size, 4)
	// A deliberately undersized buffer must not be parsed as an intact block.
	var buf [4]byte
	n, err := boundedSysctl(mib, unsafe.Pointer(&buf[0]), uintptr(len(buf)))
	if err == nil {
		require.Equal(t, len(buf), n)
		require.Empty(t, parseDarwinArgvZero(buf[:n], 8))
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Empty(t, darwinProgramName(ctx, os.Getpid(), 8))
}

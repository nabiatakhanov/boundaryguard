//go:build !windows

package main

import (
	"os/exec"
	"runtime"
	"syscall"
)

func waitMeasured(cmd *exec.Cmd) (uint64, error) {
	err := cmd.Wait()
	if cmd.ProcessState == nil {
		return 0, err
	}
	usage := cmd.ProcessState.SysUsage().(*syscall.Rusage)
	peak := uint64(usage.Maxrss)
	if runtime.GOOS != "darwin" {
		peak *= 1024
	}
	return peak, err
}

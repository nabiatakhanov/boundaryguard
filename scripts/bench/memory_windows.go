//go:build windows

package main

import (
	"os/exec"
	"syscall"
	"time"
	"unsafe"
)

func waitMeasured(cmd *exec.Cmd) (uint64, error) {
	kernel := syscall.NewLazyDLL("kernel32.dll")
	psapi := syscall.NewLazyDLL("psapi.dll")
	open := kernel.NewProc("OpenProcess")
	closeHandle := kernel.NewProc("CloseHandle")
	memory := psapi.NewProc("GetProcessMemoryInfo")
	h, _, openErr := open.Call(0x0400|0x0010, 0, uintptr(cmd.Process.Pid))
	if h == 0 {
		_ = cmd.Wait()
		return 0, openErr
	}
	defer closeHandle.Call(h)
	// PROCESS_MEMORY_COUNTERS, SIZE_T fields match native pointer width.
	type counters struct {
		Size, PageFaults                                                                                             uint32
		PeakWorkingSet, WorkingSet, PeakPagedPool, PagedPool, PeakNonPagedPool, NonPagedPool, Pagefile, PeakPagefile uintptr
	}
	var c counters
	c.Size = uint32(unsafe.Sizeof(c))
	var peak uint64
	read := func() error {
		ok, _, e := memory.Call(h, uintptr(unsafe.Pointer(&c)), uintptr(c.Size))
		if ok == 0 {
			return e
		}
		if uint64(c.PeakWorkingSet) > peak {
			peak = uint64(c.PeakWorkingSet)
		}
		return nil
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			if e := read(); e != nil && peak == 0 {
				return 0, e
			}
			return peak, err
		case <-ticker.C:
			if e := read(); e != nil {
				err := <-done
				if err != nil {
					return peak, err
				}
				return peak, e
			}
		}
	}
}

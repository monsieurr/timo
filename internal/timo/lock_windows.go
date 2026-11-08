//go:build windows

package timo

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	kernel32     = syscall.NewLazyDLL("kernel32.dll")
	lockFileEx   = kernel32.NewProc("LockFileEx")
	unlockFileEx = kernel32.NewProc("UnlockFileEx")
)

func lockFile(f *os.File) error {
	var overlap syscall.Overlapped
	r, _, err := lockFileEx.Call(f.Fd(), 0x2, 0, 1, 0, uintptr(unsafe.Pointer(&overlap)))
	runtime.KeepAlive(f)
	if r == 0 {
		return fmt.Errorf("LockFileEx: %w", err)
	}
	return nil
}
func unlockFile(f *os.File) error {
	var overlap syscall.Overlapped
	r, _, err := unlockFileEx.Call(f.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&overlap)))
	runtime.KeepAlive(f)
	if r == 0 {
		return fmt.Errorf("UnlockFileEx: %w", err)
	}
	return nil
}

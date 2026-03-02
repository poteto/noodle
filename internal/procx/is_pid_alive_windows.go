//go:build windows

package procx

import "syscall"

const (
	processStillActive  = 259
	windowsAccessDenied = 5
)

// IsPIDAlive checks process liveness via OpenProcess/GetExitCodeProcess.
// Access denied means the process exists but is owned by another user/system.
func IsPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		errno, ok := err.(syscall.Errno)
		return ok && errno == windowsAccessDenied
	}
	defer syscall.CloseHandle(handle)

	var exitCode uint32
	if err := syscall.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false
	}
	return exitCode == processStillActive
}

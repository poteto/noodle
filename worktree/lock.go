package worktree

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func (a *App) mergeLockPath() string {
	return filepath.Join(a.Root, ".worktrees", ".merge-lock")
}

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrPermission) {
		return true
	}
	return false
}

func readMergeLockPID(lockPath string) (int, error) {
	content, err := os.ReadFile(lockPath)
	if err != nil {
		return 0, err
	}
	firstLine := strings.TrimSpace(strings.SplitN(string(content), "\n", 2)[0])
	if firstLine == "" {
		return 0, fmt.Errorf("invalid lock format")
	}
	return strconv.Atoi(firstLine)
}

func writeMergeLock(lockPath string) error {
	tempPath := fmt.Sprintf("%s.%d.%d.tmp", lockPath, os.Getpid(), time.Now().UnixNano())
	f, err := os.OpenFile(tempPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(f, "%d\n%d", os.Getpid(), time.Now().Unix()); err != nil {
		f.Close()
		_ = os.Remove(tempPath)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	if err := os.Link(tempPath, lockPath); err != nil {
		_ = os.Remove(tempPath)
		if os.IsExist(err) {
			return os.ErrExist
		}
		return err
	}
	_ = os.Remove(tempPath)
	return nil
}

const (
	defaultMergeLockTimeout         = 300 * time.Second
	liveProcessLockRetryInterval    = 250 * time.Millisecond
	staleOrMissingLockRetryInterval = 100 * time.Millisecond
)

func sleepUntilMergeLockRetry(deadline time.Time, interval time.Duration) {
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return
	}
	if interval > remaining {
		interval = remaining
	}
	time.Sleep(interval)
}

func (a *App) acquireMergeLock() error {
	lockPath := a.mergeLockPath()
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return fmt.Errorf("failed to create merge lock directory: %w", err)
	}

	timeout := a.MergeLockTimeout
	if timeout == 0 {
		timeout = defaultMergeLockTimeout
	}
	deadline := time.Now().Add(timeout)
	waited := false

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf(
				"timed out waiting for merge lock (%s). Remove %s if stale.",
				timeout, lockPath,
			)
		}

		if err := writeMergeLock(lockPath); err == nil {
			if waited {
				a.info("Merge lock acquired.")
			}
			return nil
		} else if !os.IsExist(err) {
			return fmt.Errorf("failed to acquire merge lock: %w", err)
		}

		pid, err := readMergeLockPID(lockPath)
		if err != nil {
			if os.IsNotExist(err) {
				sleepUntilMergeLockRetry(deadline, staleOrMissingLockRetryInterval)
				continue
			}
			a.warnf("WARNING: removing stale merge lock %s (invalid lock format)\n", lockPath)
			if rmErr := os.Remove(lockPath); rmErr != nil && !os.IsNotExist(rmErr) {
				return fmt.Errorf("failed to remove stale merge lock: %w", rmErr)
			}
			sleepUntilMergeLockRetry(deadline, staleOrMissingLockRetryInterval)
			continue
		}

		if !isProcessAlive(pid) {
			a.warnf("WARNING: removing stale merge lock %s (PID %d)\n", lockPath, pid)
			currentPID, readErr := readMergeLockPID(lockPath)
			if readErr != nil {
				if !os.IsNotExist(readErr) {
					return fmt.Errorf("failed to verify stale merge lock: %w", readErr)
				}
			} else if currentPID == pid {
				if rmErr := os.Remove(lockPath); rmErr != nil && !os.IsNotExist(rmErr) {
					return fmt.Errorf("failed to remove stale merge lock: %w", rmErr)
				}
			}
			sleepUntilMergeLockRetry(deadline, staleOrMissingLockRetryInterval)
			continue
		}

		if !waited {
			a.info(fmt.Sprintf("Waiting for merge lock (held by PID %d)...", pid))
			waited = true
		}
		sleepUntilMergeLockRetry(deadline, liveProcessLockRetryInterval)
	}
}

func (a *App) releaseMergeLock() {
	lockPath := a.mergeLockPath()
	pid, err := readMergeLockPID(lockPath)
	if err != nil {
		return
	}
	if pid != os.Getpid() {
		return
	}
	_ = os.Remove(lockPath)
}

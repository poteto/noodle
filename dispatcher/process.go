package dispatcher

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ProcessHandle wraps an os/exec.Cmd with lifecycle primitives for child
// process management. It replaces what the tmux dispatcher provided: process isolation,
// liveness checking, and graceful shutdown.
type ProcessHandle struct {
	cmd *exec.Cmd

	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	done     chan struct{}
	doneOnce sync.Once

	mu       sync.Mutex
	exitCode int
	exited   bool
}

// processMetadata is written to process.json for crash recovery.
type processMetadata struct {
	PID       int       `json:"pid"`
	SessionID string    `json:"session_id"`
	StartedAt time.Time `json:"started_at"`
}

// StartProcess spawns a child process with process group isolation and
// returns a handle for lifecycle management.
func StartProcess(cmd *exec.Cmd) (*ProcessHandle, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start process: %w", err)
	}

	h := &ProcessHandle{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		done:   make(chan struct{}),
	}

	go h.wait()
	return h, nil
}

// Stdin returns the write end of the child's stdin pipe.
func (h *ProcessHandle) Stdin() io.WriteCloser { return h.stdin }

// Stdout returns the read end of the child's stdout pipe.
func (h *ProcessHandle) Stdout() io.ReadCloser { return h.stdout }

// Stderr returns the read end of the child's stderr pipe.
func (h *ProcessHandle) Stderr() io.ReadCloser { return h.stderr }

// Done returns a channel that is closed when the process exits.
func (h *ProcessHandle) Done() <-chan struct{} { return h.done }

// PID returns the process ID, or 0 if the process hasn't started.
func (h *ProcessHandle) PID() int {
	if h.cmd.Process == nil {
		return 0
	}
	return h.cmd.Process.Pid
}

// ExitCode returns the exit code and whether the process has exited.
func (h *ProcessHandle) ExitCode() (int, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.exitCode, h.exited
}

// Kill sends SIGTERM to the process group, waits up to 5 seconds,
// then sends SIGKILL if the process is still running.
func (h *ProcessHandle) Kill() error {
	if h.cmd.Process == nil {
		return nil
	}
	pgid, err := syscall.Getpgid(h.cmd.Process.Pid)
	if err != nil {
		// Process already gone.
		return nil
	}

	_ = syscall.Kill(-pgid, syscall.SIGTERM)

	select {
	case <-h.done:
		return nil
	case <-time.After(5 * time.Second):
	}

	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	<-h.done
	return nil
}

func (h *ProcessHandle) wait() {
	err := h.cmd.Wait()
	h.mu.Lock()
	h.exited = true
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			h.exitCode = exitErr.ExitCode()
		} else {
			h.exitCode = -1
		}
	}
	h.mu.Unlock()
	h.doneOnce.Do(func() { close(h.done) })
}

// WriteProcessMetadata writes process.json to the session directory.
func WriteProcessMetadata(sessionDir, sessionID string, pid int, startedAt time.Time) error {
	sessionDir = strings.TrimSpace(sessionDir)
	if sessionDir == "" {
		return fmt.Errorf("session directory not set")
	}
	payload, err := json.Marshal(processMetadata{
		PID:       pid,
		SessionID: sessionID,
		StartedAt: startedAt.UTC(),
	})
	if err != nil {
		return fmt.Errorf("encode process metadata: %w", err)
	}
	path := filepath.Join(sessionDir, "process.json")
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write process metadata: %w", err)
	}
	return nil
}

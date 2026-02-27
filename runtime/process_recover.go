package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/poteto/noodle/dispatcher"
)

// processRuntime adds PID-based session recovery to the base DispatcherRuntime.
type processRuntime struct {
	*DispatcherRuntime
}

// NewProcessRuntime adapts a process dispatcher to the Runtime interface with
// PID-based session recovery.
func NewProcessRuntime(d dispatcher.Dispatcher, runtimeDir string, maxConcurrent int) Runtime {
	r := NewDispatcherRuntime("process", d, runtimeDir)
	r.SetMaxConcurrent(maxConcurrent)
	return &processRuntime{DispatcherRuntime: r}
}

func (p *processRuntime) Recover(_ context.Context) ([]RecoveredSession, error) {
	sessionsDir := filepath.Join(p.runtimeDir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var recovered []RecoveredSession
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sessionID := entry.Name()
		sessionDir := filepath.Join(sessionsDir, sessionID)

		pid, err := readRecoverPID(sessionDir)
		if err != nil {
			// No process.json — skip.
			continue
		}

		metaPath := filepath.Join(sessionDir, "meta.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		if !strings.Contains(string(data), `"status":"running"`) {
			continue
		}

		if !recoverPIDAlive(pid) {
			// Dead process — update meta.json status.
			updated := strings.Replace(string(data), `"status":"running"`, `"status":"exited"`, 1)
			_ = os.WriteFile(metaPath, []byte(updated), 0o644)
			continue
		}

		target := ReadSessionTarget(filepath.Join(sessionDir, "prompt.txt"))
		recovered = append(recovered, RecoveredSession{
			OrderID:       target,
			SessionHandle: &recoveredSessionHandle{id: sessionID, status: "running"},
			RuntimeName:   p.name,
			Reason:        "live PID found",
		})
	}

	return recovered, nil
}

// readRecoverPID reads the PID from a session's process.json.
func readRecoverPID(sessionDir string) (int, error) {
	data, err := os.ReadFile(filepath.Join(sessionDir, "process.json"))
	if err != nil {
		return 0, err
	}
	var meta struct {
		PID int `json:"pid"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return 0, fmt.Errorf("parse process.json: %w", err)
	}
	if meta.PID <= 0 {
		return 0, fmt.Errorf("invalid PID %d", meta.PID)
	}
	return meta.PID, nil
}

// recoverPIDAlive checks whether a process is alive using kill(pid, 0).
func recoverPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	if errors.Is(err, syscall.EPERM) {
		return true
	}
	return false
}

// recoveredSessionHandle represents a session discovered during crash recovery.
type recoveredSessionHandle struct {
	id     string
	status string
}

func (s *recoveredSessionHandle) ID() string          { return s.id }
func (s *recoveredSessionHandle) Status() string      { return s.status }
func (s *recoveredSessionHandle) TotalCost() float64  { return 0 }
func (s *recoveredSessionHandle) Kill() error         { return nil }
func (s *recoveredSessionHandle) VerdictPath() string { return "" }
func (s *recoveredSessionHandle) Controller() AgentController {
	return dispatcher.NoopController()
}

func (s *recoveredSessionHandle) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// scheduleOrderID is the well-known order ID for the scheduler session.
const scheduleOrderID = "schedule"

// Prompt parsing patterns for extracting order IDs from session prompts.
var (
	promptOrderRegexp    = regexp.MustCompile(`(?im)^\[order:([^\]]+)\]`)
	promptItemRegexp     = regexp.MustCompile(`(?im)^work backlog item\s+([^\r\n]+)$`)
	schedulePromptRegexp = regexp.MustCompile(`(?im)^\s*use skill\([^)]+\)\s+to refresh .+from \.noodle/mise\.json\.`)
)

// ReadSessionTarget extracts the order ID from a session's prompt file.
func ReadSessionTarget(promptPath string) string {
	data, err := os.ReadFile(promptPath)
	if err != nil {
		return ""
	}

	// Try new format first: [order:ID]
	orderMatches := promptOrderRegexp.FindStringSubmatch(string(data))
	if len(orderMatches) == 2 {
		return strings.TrimSpace(orderMatches[1])
	}

	// Old format: Work backlog item <id>
	matches := promptItemRegexp.FindStringSubmatch(string(data))
	if len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}

	if schedulePromptRegexp.Match(data) {
		return scheduleOrderID
	}
	return ""
}

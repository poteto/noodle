package runtime

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/internal/procx"
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

		pid, err := procx.ReadPIDFile(filepath.Join(sessionDir, "process.json"))
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

		if !procx.IsPIDAlive(pid) {
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

// recoveredSessionHandle represents a session discovered during crash recovery.
type recoveredSessionHandle struct {
	id     string
	status string
}

func (s *recoveredSessionHandle) ID() string          { return s.id }
func (s *recoveredSessionHandle) Status() string      { return s.status }
func (s *recoveredSessionHandle) TotalCost() float64  { return 0 }
func (s *recoveredSessionHandle) Terminate() error    { return nil }
func (s *recoveredSessionHandle) ForceKill() error    { return nil }
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
	schedulePromptRegexp = regexp.MustCompile(`(?im)^\s*use skill\([^)]+\)\s+to refresh .+from \.noodle/mise\.json\.`)
)

// ReadSessionTarget extracts the order ID from a session's prompt file.
func ReadSessionTarget(promptPath string) string {
	data, err := os.ReadFile(promptPath)
	if err != nil {
		return ""
	}

	orderMatches := promptOrderRegexp.FindStringSubmatch(string(data))
	if len(orderMatches) == 2 {
		return strings.TrimSpace(orderMatches[1])
	}

	if schedulePromptRegexp.Match(data) {
		return scheduleOrderID
	}
	return ""
}

package runtime

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/poteto/noodle/dispatcher"
)

// tmuxRuntime adds tmux-specific session recovery to the base DispatcherRuntime.
type tmuxRuntime struct {
	*DispatcherRuntime
}

// scheduleOrderID is the well-known order ID for the scheduler session.
const scheduleOrderID = "schedule"

func (t *tmuxRuntime) Recover(_ context.Context) ([]RecoveredSession, error) {
	sessionsDir := filepath.Join(t.runtimeDir, "sessions")
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

	alive := map[string]struct{}{}
	for _, name := range ListTmuxSessions() {
		alive[name] = struct{}{}
	}
	knownTmux := map[string]struct{}{}
	var recovered []RecoveredSession

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sessionID := entry.Name()
		sessionName := TmuxSessionName(sessionID)
		knownTmux[sessionName] = struct{}{}

		metaPath := filepath.Join(sessionsDir, sessionID, "meta.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		if strings.Contains(string(data), `"status":"running"`) {
			if _, ok := alive[sessionName]; !ok {
				updated := strings.Replace(string(data), `"status":"running"`, `"status":"exited"`, 1)
				_ = os.WriteFile(metaPath, []byte(updated), 0o644)
				continue
			}
			target := ReadSessionTarget(filepath.Join(sessionsDir, sessionID, "prompt.txt"))
			recovered = append(recovered, RecoveredSession{
				OrderID:       target,
				SessionHandle: &recoveredSessionHandle{id: sessionID, status: "running"},
				RuntimeName:   t.name,
				Reason:        "live tmux session found",
			})
		}
	}

	// Kill orphaned tmux sessions (alive but no matching session dir).
	for name := range alive {
		if !strings.HasPrefix(name, "noodle-") {
			continue
		}
		if _, ok := knownTmux[name]; ok {
			continue
		}
		_ = KillTmuxSession(name)
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
func (s *recoveredSessionHandle) Kill() error         { return nil }
func (s *recoveredSessionHandle) VerdictPath() string       { return "" }
func (s *recoveredSessionHandle) Controller() AgentController { return dispatcher.NoopController() }

func (s *recoveredSessionHandle) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// ListTmuxSessions returns the names of all active tmux sessions.
func ListTmuxSessions() []string {
	out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	outList := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		outList = append(outList, line)
	}
	return outList
}

// KillTmuxSession kills a tmux session by name.
func KillTmuxSession(name string) error {
	return exec.Command("tmux", "kill-session", "-t", name).Run()
}

// TmuxSessionName converts a session ID to its tmux session name.
func TmuxSessionName(sessionID string) string {
	return "noodle-" + SanitizeSessionToken(sessionID, "cook")
}

// SanitizeSessionToken normalizes a string for use in tmux session names.
func SanitizeSessionToken(value string, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = fallback
	}
	var out strings.Builder
	lastHyphen := false
	for _, r := range value {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			out.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			out.WriteByte('-')
			lastHyphen = true
		}
	}
	result := strings.Trim(out.String(), "-")
	if result == "" {
		result = fallback
	}
	if len(result) > 48 {
		result = strings.Trim(result[:48], "-")
	}
	if result == "" {
		return fallback
	}
	return result
}

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

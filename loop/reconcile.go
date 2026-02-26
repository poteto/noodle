package loop

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/monitor"
)

func (l *Loop) reconcile(ctx context.Context) error {
	if err := l.loadPendingReview(); err != nil {
		return err
	}
	// Prune pending reviews for orders that no longer exist in orders.json.
	// This handles the crash window between advancing orders.json and updating
	// pending-review.json (finding #5).
	if err := l.reconcilePendingReview(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(l.runtimeDir, "sessions"), 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Join(l.runtimeDir, "sessions"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	alive := map[string]struct{}{}
	for _, name := range listTmuxSessions() {
		alive[name] = struct{}{}
	}
	knownTmux := map[string]struct{}{}
	l.adoptedTargets = map[string]string{}
	l.adoptedSessions = l.adoptedSessions[:0]

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sessionID := entry.Name()
		sessionName := tmuxSessionName(sessionID)
		knownTmux[sessionName] = struct{}{}

		metaPath := filepath.Join(l.runtimeDir, "sessions", sessionID, "meta.json")
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
			target := readSessionTarget(filepath.Join(l.runtimeDir, "sessions", sessionID, "prompt.txt"))
			if target != "" {
				l.adoptedTargets[target] = sessionID
			}
			l.adoptedSessions = append(l.adoptedSessions, sessionID)
		}
	}

	for name := range alive {
		if !strings.HasPrefix(name, "noodle-") {
			continue
		}
		if _, ok := knownTmux[name]; ok {
			continue
		}
		_ = killTmuxSession(name)
	}

	if len(l.adoptedSessions) > 0 {
		tickets := monitor.NewEventTicketMaterializer(l.runtimeDir)
		_ = tickets.Materialize(ctx, l.adoptedSessions)
	}
	return nil
}

func listTmuxSessions() []string {
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

// New format: [order:ID] Work on plan: ... or [order:ID] Work backlog item ID
var promptOrderRegexp = regexp.MustCompile(`(?im)^\[order:([^\]]+)\]`)

// Old format: Work backlog item <id>
var promptItemRegexp = regexp.MustCompile(`(?im)^work backlog item\s+([^\r\n]+)$`)
var schedulePromptRegexp = regexp.MustCompile(`(?im)^\s*use skill\([^)]+\)\s+to refresh .+from \.noodle/mise\.json\.`)

func readSessionTarget(promptPath string) string {
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

func (l *Loop) refreshAdoptedTargets() {
	if len(l.adoptedTargets) == 0 {
		return
	}
	alive := map[string]struct{}{}
	for _, name := range listTmuxSessions() {
		alive[name] = struct{}{}
	}
	nextTargets := make(map[string]string, len(l.adoptedTargets))
	nextSessions := make([]string, 0, len(l.adoptedTargets))
	for target, sessionID := range l.adoptedTargets {
		metaPath := filepath.Join(l.runtimeDir, "sessions", sessionID, "meta.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		if !strings.Contains(string(data), `"status":"running"`) {
			continue
		}
		if _, ok := alive[tmuxSessionName(sessionID)]; !ok {
			continue
		}
		nextTargets[target] = sessionID
		nextSessions = append(nextSessions, sessionID)
	}
	l.adoptedTargets = nextTargets
	l.adoptedSessions = nextSessions
}

func tmuxSessionName(sessionID string) string {
	return "noodle-" + sanitizeSessionToken(sessionID, "cook")
}

type adoptedSession struct {
	id     string
	status string
}

func (s *adoptedSession) ID() string {
	return s.id
}

func (s *adoptedSession) Status() string {
	return s.status
}

func (s *adoptedSession) Events() <-chan dispatcher.SessionEvent {
	ch := make(chan dispatcher.SessionEvent)
	close(ch)
	return ch
}

func (s *adoptedSession) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (s *adoptedSession) TotalCost() float64 {
	return 0
}

func (s *adoptedSession) Kill() error {
	return nil
}

func killTmuxSession(name string) error {
	return exec.Command("tmux", "kill-session", "-t", name).Run()
}

func sanitizeSessionToken(value string, fallback string) string {
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

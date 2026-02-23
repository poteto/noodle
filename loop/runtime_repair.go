package loop

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/poteto/noodle/spawner"
)

const maxRuntimeRepairAttempts = 3

func (l *Loop) handleRuntimeIssue(ctx context.Context, scope string, err error, warnings []string) error {
	issue := runtimeIssue{
		Scope:    strings.TrimSpace(scope),
		Message:  strings.TrimSpace(issueMessage(err, warnings)),
		Warnings: normalizeWarnings(warnings),
		Stack:    strings.TrimSpace(string(debug.Stack())),
	}
	return l.ensureRuntimeRepair(ctx, issue)
}

func issueMessage(err error, warnings []string) string {
	if err != nil {
		return err.Error()
	}
	if len(warnings) == 0 {
		return "unknown runtime issue"
	}
	return strings.Join(normalizeWarnings(warnings), "; ")
}

func normalizeWarnings(warnings []string) []string {
	if len(warnings) == 0 {
		return nil
	}
	out := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		warning = strings.TrimSpace(warning)
		if warning == "" {
			continue
		}
		out = append(out, warning)
	}
	sort.Strings(out)
	return out
}

func (l *Loop) ensureRuntimeRepair(ctx context.Context, issue runtimeIssue) error {
	if l.runtimeRepairInFlight != nil {
		return nil
	}
	if l.adoptRunningRuntimeRepair(issue) {
		return nil
	}
	fingerprint := runtimeIssueFingerprint(issue)
	attempt := l.runtimeRepairAttempts[fingerprint] + 1
	if attempt > maxRuntimeRepairAttempts {
		return fmt.Errorf(
			"runtime issue unresolved after %d repair attempt(s) (%s): %s",
			maxRuntimeRepairAttempts,
			nonEmpty(issue.Scope, "runtime"),
			nonEmpty(issue.Message, "unknown runtime issue"),
		)
	}

	stateBefore := l.state
	if stateBefore == StateRunning {
		l.state = StatePaused
	}

	session, err := l.spawnRuntimeRepair(ctx, issue, attempt)
	if err != nil {
		return fmt.Errorf("runtime repair unavailable (%s): %w", nonEmpty(issue.Scope, "runtime"), err)
	}

	l.runtimeRepairAttempts[fingerprint] = attempt
	l.runtimeRepairInFlight = &runtimeRepairState{
		Fingerprint: fingerprint,
		Issue:       issue,
		Attempt:     attempt,
		SessionID:   session.ID(),
		Session:     session,
		StateBefore: stateBefore,
	}
	return nil
}

func (l *Loop) adoptRunningRuntimeRepair(issue runtimeIssue) bool {
	sessionID := l.findRunningRuntimeRepairSessionID()
	if sessionID == "" {
		return false
	}
	stateBefore := l.state
	if stateBefore == StateRunning {
		l.state = StatePaused
	}
	l.runtimeRepairInFlight = &runtimeRepairState{
		Fingerprint: runtimeIssueFingerprint(issue),
		Issue:       issue,
		Attempt:     l.runtimeRepairAttempts[runtimeIssueFingerprint(issue)],
		SessionID:   sessionID,
		StateBefore: stateBefore,
	}
	return true
}

func (l *Loop) findRunningRuntimeRepairSessionID() string {
	entries, err := os.ReadDir(filepath.Join(l.runtimeDir, "sessions"))
	if err != nil {
		return ""
	}
	latest := ""
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaPath := filepath.Join(l.runtimeDir, "sessions", entry.Name(), "meta.json")
		data, readErr := os.ReadFile(metaPath)
		if readErr != nil {
			continue
		}
		var payload struct {
			SessionID string `json:"session_id"`
			Status    string `json:"status"`
		}
		if unmarshalErr := json.Unmarshal(data, &payload); unmarshalErr != nil {
			continue
		}
		sessionID := strings.TrimSpace(payload.SessionID)
		if sessionID == "" {
			sessionID = strings.TrimSpace(entry.Name())
		}
		status := strings.ToLower(strings.TrimSpace(payload.Status))
		if status != "running" && status != "spawning" && status != "stuck" {
			continue
		}
		if !strings.HasPrefix(sessionID, "repair-runtime-") {
			continue
		}
		if latest == "" || sessionID > latest {
			latest = sessionID
		}
	}
	return latest
}

func runtimeIssueFingerprint(issue runtimeIssue) string {
	builder := strings.Builder{}
	builder.WriteString(strings.TrimSpace(issue.Scope))
	builder.WriteByte('|')
	builder.WriteString(strings.TrimSpace(issue.Message))
	for _, warning := range normalizeWarnings(issue.Warnings) {
		builder.WriteByte('|')
		builder.WriteString(strings.TrimSpace(warning))
	}
	hash := sha1.Sum([]byte(builder.String()))
	return hex.EncodeToString(hash[:8])
}

func (l *Loop) advanceRuntimeRepair(ctx context.Context) error {
	if l.runtimeRepairInFlight == nil {
		return nil
	}

	if l.runtimeRepairInFlight.Session != nil {
		select {
		case <-l.runtimeRepairInFlight.Session.Done():
		default:
			if l.runtimeRepairInFlight.StateBefore == StateRunning {
				l.state = StatePaused
			}
			return nil
		}
	}

	inFlight := l.runtimeRepairInFlight
	l.runtimeRepairInFlight = nil

	status := ""
	if inFlight.Session != nil {
		status = strings.ToLower(strings.TrimSpace(inFlight.Session.Status()))
	} else if strings.TrimSpace(inFlight.SessionID) != "" {
		currentStatus, ok, err := l.readSessionStatus(inFlight.SessionID)
		if err != nil {
			return err
		}
		if !ok {
			l.state = inFlight.StateBefore
			return nil
		}
		switch currentStatus {
		case "running", "spawning", "stuck":
			l.runtimeRepairInFlight = inFlight
			if inFlight.StateBefore == StateRunning {
				l.state = StatePaused
			}
			return nil
		}
		status = currentStatus
	}
	if status == "completed" {
		l.state = inFlight.StateBefore
		return nil
	}
	if status == "exited" {
		return fmt.Errorf(
			"runtime repair session %s exited before completion",
			nonEmpty(strings.TrimSpace(inFlight.SessionID), "unknown-session"),
		)
	}

	retryIssue := inFlight.Issue
	retryIssue.Stack = strings.TrimSpace(strings.Join([]string{
		strings.TrimSpace(retryIssue.Stack),
		fmt.Sprintf(
			"repair session %s ended with status %s",
			nonEmpty(strings.TrimSpace(inFlight.SessionID), "unknown-session"),
			nonEmpty(status, "unknown"),
		),
	}, "\n"))
	return l.ensureRuntimeRepair(ctx, retryIssue)
}

func (l *Loop) spawnRuntimeRepair(ctx context.Context, issue runtimeIssue, attempt int) (spawner.Session, error) {
	name := fmt.Sprintf("repair-runtime-%s-%d", time.Now().UTC().Format("20060102-150405"), attempt)
	if err := l.deps.Worktree.Create(name); err != nil {
		return nil, fmt.Errorf("create repair worktree: %w", err)
	}

	worktreePath := filepath.Join(l.projectDir, ".worktrees", name)
	request := spawner.SpawnRequest{
		Name:         name,
		Prompt:       buildRuntimeRepairPrompt(issue, attempt),
		Provider:     nonEmpty(l.config.Routing.Defaults.Provider, "claude"),
		Model:        nonEmpty(l.config.Routing.Defaults.Model, "claude-sonnet-4-6"),
		Skill:        l.runtimeRepairSkill(),
		WorktreePath: worktreePath,
	}
	session, err := l.deps.Spawner.Spawn(ctx, request)
	if err != nil {
		_ = l.deps.Worktree.Cleanup(name, true)
		return nil, err
	}
	return session, nil
}

func (l *Loop) runtimeRepairSkill() string {
	if _, ok := l.registry.ByKey("oops"); ok {
		return "oops"
	}
	return "debugging"
}

func buildRuntimeRepairPrompt(issue runtimeIssue, attempt int) string {
	var b strings.Builder
	b.WriteString("Noodle runtime self-healing task.\n")
	b.WriteString("A runtime infrastructure issue blocked scheduling. Repair the root cause so `noodle start` can continue.\n")
	b.WriteString("Keep the fix minimal and avoid unrelated refactors.\n\n")
	fmt.Fprintf(&b, "Scope: %s\n", nonEmpty(strings.TrimSpace(issue.Scope), "runtime"))
	fmt.Fprintf(&b, "Attempt: %d\n", attempt)
	fmt.Fprintf(&b, "Error: %s\n", nonEmpty(strings.TrimSpace(issue.Message), "unknown runtime issue"))
	if len(issue.Warnings) > 0 {
		b.WriteString("Warnings:\n")
		for _, warning := range issue.Warnings {
			fmt.Fprintf(&b, "- %s\n", warning)
		}
	}
	if strings.TrimSpace(issue.Stack) != "" {
		b.WriteString("\nStack trace:\n")
		b.WriteString(issue.Stack)
		b.WriteString("\n")
	}
	b.WriteString("\nRequired verification:\n")
	b.WriteString("- Re-run the failing flow and confirm the runtime issue no longer appears.\n")
	b.WriteString("- If Go files changed, run `go test ./...`.\n")
	b.WriteString("- Return a short summary of files changed and verification results.\n")
	return b.String()
}

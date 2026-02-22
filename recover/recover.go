package recover

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/poteto/noodle/event"
)

var recoverySuffixRegexp = regexp.MustCompile(`-recover-(\d+)$`)

// RecoveryInfo summarizes what happened in a failed session.
type RecoveryInfo struct {
	SessionID    string
	ExitReason   string
	LastAction   string
	FilesChanged []string
}

// ResumeContext is the prompt block injected into recovery attempts.
type ResumeContext struct {
	Attempt int
	Summary string
}

// RecoveryChainLength returns retry depth encoded in a session name.
func RecoveryChainLength(name string) int {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0
	}
	matches := recoverySuffixRegexp.FindStringSubmatch(name)
	if len(matches) != 2 {
		return 0
	}
	var value int
	_, _ = fmt.Sscanf(matches[1], "%d", &value)
	if value < 0 {
		return 0
	}
	return value
}

// NextRecoveryName appends retry suffix according to configured pattern.
func NextRecoveryName(base string, attempt int, suffixPattern string) (string, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		return "", fmt.Errorf("base name is required")
	}
	if attempt <= 0 {
		return "", fmt.Errorf("attempt must be greater than 0")
	}
	pattern := strings.TrimSpace(suffixPattern)
	if pattern == "" {
		pattern = "-recover-%d"
	}
	if !strings.Contains(pattern, "%d") {
		return "", fmt.Errorf("recovery suffix pattern must include %%d placeholder")
	}
	root := recoverySuffixRegexp.ReplaceAllString(base, "")
	return root + fmt.Sprintf(pattern, attempt), nil
}

// BuildResumeContext formats a concise retry summary for prompt injection.
func BuildResumeContext(info RecoveryInfo, attempt int, maxRetries int) ResumeContext {
	files := normalizeFiles(info.FilesChanged)
	summary := strings.Builder{}
	summary.WriteString("Recovery context:\n")
	if strings.TrimSpace(info.ExitReason) != "" {
		summary.WriteString("- Failure: ")
		summary.WriteString(strings.TrimSpace(info.ExitReason))
		summary.WriteString("\n")
	}
	if strings.TrimSpace(info.LastAction) != "" {
		summary.WriteString("- Last action: ")
		summary.WriteString(strings.TrimSpace(info.LastAction))
		summary.WriteString("\n")
	}
	if len(files) > 0 {
		summary.WriteString("- Files touched:\n")
		for _, file := range files {
			summary.WriteString("  - ")
			summary.WriteString(file)
			summary.WriteString("\n")
		}
	}
	if maxRetries > 0 {
		summary.WriteString(fmt.Sprintf("- Attempt: %d/%d\n", attempt, maxRetries))
	}
	return ResumeContext{
		Attempt: attempt,
		Summary: strings.TrimSpace(summary.String()),
	}
}

// CollectRecoveryInfo reconstructs recent context from session events.
func CollectRecoveryInfo(ctx context.Context, runtimeDir, sessionID string) (RecoveryInfo, error) {
	reader := event.NewEventReader(runtimeDir)
	records, err := reader.ReadSession(sessionID, event.EventFilter{})
	if err != nil {
		return RecoveryInfo{}, err
	}
	info := RecoveryInfo{SessionID: sessionID}
	for _, record := range records {
		switch record.Type {
		case event.EventAction:
			if action := extractPayloadString(record.Payload, "message"); action != "" {
				info.LastAction = action
			}
			info.FilesChanged = append(info.FilesChanged, extractLikelyPath(record.Payload)...)
		case event.EventStateChange:
			if reason := extractPayloadString(record.Payload, "reason"); reason != "" {
				info.ExitReason = reason
			}
		case event.EventExited:
			if outcome := extractPayloadString(record.Payload, "outcome"); outcome != "" && info.ExitReason == "" {
				info.ExitReason = outcome
			}
		}
	}
	if strings.TrimSpace(info.ExitReason) == "" {
		info.ExitReason = "session exited without explicit reason"
	}
	info.FilesChanged = normalizeFiles(info.FilesChanged)
	return info, ctx.Err()
}

func normalizeFiles(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	ordered := make([]string, 0, len(paths))
	for _, candidate := range paths {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if strings.Contains(candidate, " ") {
			continue
		}
		clean := filepath.Clean(candidate)
		if clean == "." || clean == ".." {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		ordered = append(ordered, clean)
	}
	sort.Strings(ordered)
	return ordered
}

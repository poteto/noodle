package loop

import (
	"fmt"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/stringx"
	"github.com/poteto/noodle/mise"
)

func buildCookPrompt(item QueueItem, resumePrompt string) string {
	parts := []string{fmt.Sprintf("Work backlog item %s", item.ID)}
	if strings.TrimSpace(item.Rationale) != "" {
		parts = append(parts, "Context: "+strings.TrimSpace(item.Rationale))
	}
	if strings.TrimSpace(resumePrompt) != "" {
		parts = append(parts, resumePrompt)
	}
	return strings.Join(parts, "\n\n")
}

func nonEmpty(value, fallback string) string {
	return stringx.NonEmpty(value, fallback)
}

func sanitizeName(value string) string {
	name := sanitizeToken(value)
	if name == "" {
		return "cook"
	}
	return name
}

func sanitizeToken(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	builder := strings.Builder{}
	lastHyphen := false
	for _, r := range value {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			builder.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			builder.WriteByte('-')
			lastHyphen = true
		}
	}
	name := strings.Trim(builder.String(), "-")
	return name
}

func cookBaseName(item QueueItem) string {
	idToken := sanitizeToken(item.ID)
	if idToken == "" {
		idToken = "cook"
	}

	titleToken := sanitizeToken(item.Title)
	if titleToken == "" {
		return idToken
	}

	titleToken = truncateToken(titleToken, 32)
	if titleToken == "" {
		return idToken
	}

	const maxNameLen = 64
	base := idToken + "-" + titleToken
	if len(base) <= maxNameLen {
		return base
	}

	maxTitleLen := maxNameLen - len(idToken) - 1
	if maxTitleLen <= 0 {
		return idToken
	}
	titleToken = truncateToken(titleToken, maxTitleLen)
	if titleToken == "" {
		return idToken
	}
	return idToken + "-" + titleToken
}

func truncateToken(token string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	token = strings.Trim(token, "-")
	if token == "" {
		return ""
	}
	if len(token) <= maxLen {
		return token
	}
	token = token[:maxLen]
	return strings.Trim(token, "-")
}

func (l *Loop) pollInterval() time.Duration {
	interval := strings.TrimSpace(l.config.Monitor.PollInterval)
	if interval == "" {
		return 5 * time.Second
	}
	duration, err := time.ParseDuration(interval)
	if err != nil || duration <= 0 {
		return 5 * time.Second
	}
	return duration
}

func hasActiveTicket(brief mise.Brief, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	for _, ticket := range brief.Tickets {
		if ticket.Target != target {
			continue
		}
		switch ticket.Status {
		case "active", "blocked":
			return true
		}
	}
	return false
}

func isMissingAdapter(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "not configured") || strings.Contains(text, "no such file")
}

func isWorktreeAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "worktree") && strings.Contains(text, "already exists at")
}

func findQueueItemByTarget(items []QueueItem, targetID string) (QueueItem, bool) {
	targetID = strings.TrimSpace(targetID)
	if targetID == "" {
		return QueueItem{}, false
	}
	for _, item := range items {
		if strings.TrimSpace(item.ID) == targetID {
			return item, true
		}
	}
	return QueueItem{}, false
}

func shouldRecoverMissingSyncScripts(warnings []string, queue Queue) bool {
	if len(queue.Items) > 0 {
		return false
	}
	for _, warning := range warnings {
		warning = strings.ToLower(strings.TrimSpace(warning))
		if warning == "" {
			continue
		}
		if strings.Contains(warning, "sync script missing") {
			return true
		}
	}
	return false
}

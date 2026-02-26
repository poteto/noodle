package loop

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/stringx"
	"github.com/poteto/noodle/mise"
)

func buildCookPrompt(item QueueItem, resumePrompt string) string {
	var header string
	if len(item.Plan) > 0 {
		header = fmt.Sprintf("Work on plan: %s", strings.Join(item.Plan, ", "))
	} else {
		header = fmt.Sprintf("Work backlog item %s", item.ID)
	}
	parts := []string{header}
	if skill := strings.TrimSpace(item.Skill); skill != "" {
		parts = append(parts, fmt.Sprintf("Use Skill(%s) for methodology.", skill))
	}
	if strings.TrimSpace(item.Prompt) != "" {
		parts = append(parts, strings.TrimSpace(item.Prompt))
	}
	if strings.TrimSpace(item.Rationale) != "" {
		parts = append(parts, "Context: "+strings.TrimSpace(item.Rationale))
	}
	if strings.TrimSpace(resumePrompt) != "" {
		parts = append(parts, resumePrompt)
	}
	return strings.Join(parts, "\n\n")
}

// titleFromPrompt derives a short title from the first few words of a prompt.
func titleFromPrompt(prompt string, maxWords int) string {
	words := strings.Fields(prompt)
	if len(words) <= maxWords {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:maxWords], " ") + "..."
}

func joinPromptParts(parts ...string) string {
	var out []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, strings.TrimSpace(p))
		}
	}
	return strings.Join(out, "\n\n")
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

// hasSyncWarnings returns true if any warning indicates a sync script problem.
func hasSyncWarnings(warnings []string) bool {
	for _, warning := range warnings {
		warning = strings.ToLower(strings.TrimSpace(warning))
		if warning == "" {
			continue
		}
		if strings.Contains(warning, "sync script missing") || strings.Contains(warning, "sync script failed") {
			return true
		}
	}
	return false
}

// worktreeResumeContext checks if a worktree has commits ahead of main.
// Returns a resume hint for the agent prompt, or empty string if the
// worktree is fresh.
func worktreeResumeContext(worktreePath, name string) string {
	out, err := exec.Command("git", "-C", worktreePath, "log", "main..HEAD", "--oneline").Output()
	if err != nil {
		return ""
	}
	lines := strings.TrimSpace(string(out))
	if lines == "" {
		return ""
	}
	commits := strings.Split(lines, "\n")
	return fmt.Sprintf(
		"You are resuming work in worktree %q which has %d prior commit(s):\n%s\n\nReview what was done and continue with the remaining work.",
		name, len(commits), lines,
	)
}

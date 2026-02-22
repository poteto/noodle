package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/poteto/noodle/event"
)

func loadSnapshot(runtimeDir string, now time.Time) (Snapshot, error) {
	sessions, err := readSessions(runtimeDir)
	if err != nil {
		return Snapshot{}, err
	}
	queue, err := readQueue(filepath.Join(runtimeDir, "queue.json"))
	if err != nil {
		return Snapshot{}, err
	}

	eventsBySession := make(map[string][]EventLine, len(sessions))
	reader := event.NewEventReader(runtimeDir)
	for i := range sessions {
		events, err := reader.ReadSession(sessions[i].ID, event.EventFilter{})
		if err != nil {
			return Snapshot{}, err
		}
		lines := mapEventLines(events)
		eventsBySession[sessions[i].ID] = lines
		if sessions[i].CurrentAction == "" && len(lines) > 0 {
			sessions[i].CurrentAction = lines[len(lines)-1].Body
		}
	}

	active := make([]Session, 0, len(sessions))
	recent := make([]Session, 0, len(sessions))
	totalCost := 0.0
	loopState := "running"
	for _, session := range sessions {
		totalCost += session.TotalCostUSD
		loopState = pickLoopState(loopState, session.LoopState)
		if isActiveStatus(session.Status) {
			active = append(active, session)
		} else {
			recent = append(recent, session)
		}
	}

	sort.SliceStable(active, func(i, j int) bool {
		if active[i].LastActivity.Equal(active[j].LastActivity) {
			return active[i].ID < active[j].ID
		}
		return active[i].LastActivity.After(active[j].LastActivity)
	})
	sort.SliceStable(recent, func(i, j int) bool {
		if recent[i].LastActivity.Equal(recent[j].LastActivity) {
			return recent[i].ID < recent[j].ID
		}
		return recent[i].LastActivity.After(recent[j].LastActivity)
	})

	return Snapshot{
		UpdatedAt:       now.UTC(),
		LoopState:       loopState,
		Sessions:        sessions,
		Active:          active,
		Recent:          recent,
		Queue:           queue,
		EventsBySession: eventsBySession,
		TotalCostUSD:    totalCost,
	}, nil
}

func readSessions(runtimeDir string) ([]Session, error) {
	path := filepath.Join(runtimeDir, "sessions")
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read sessions directory: %w", err)
	}

	sessions := make([]Session, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sessionID := strings.TrimSpace(entry.Name())
		if sessionID == "" {
			continue
		}
		metaPath := filepath.Join(path, sessionID, "meta.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read session meta %s: %w", sessionID, err)
		}

		var raw struct {
			SessionID             string  `json:"session_id"`
			Status                string  `json:"status"`
			Provider              string  `json:"provider"`
			Model                 string  `json:"model"`
			TotalCostUSD          float64 `json:"total_cost_usd"`
			DurationSeconds       int64   `json:"duration_seconds"`
			LastActivity          string  `json:"last_activity"`
			CurrentAction         string  `json:"current_action"`
			Health                string  `json:"health"`
			ContextWindowUsagePct float64 `json:"context_window_usage_pct"`
			RetryCount            int     `json:"retry_count"`
			IdleSeconds           int64   `json:"idle_seconds"`
			StuckThresholdSeconds int64   `json:"stuck_threshold_seconds"`
			LoopState             string  `json:"loop_state"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse session meta %s: %w", sessionID, err)
		}

		if strings.TrimSpace(raw.SessionID) != "" {
			sessionID = strings.TrimSpace(raw.SessionID)
		}
		lastActivity := parseTime(raw.LastActivity)
		status := strings.ToLower(strings.TrimSpace(raw.Status))
		health := deriveHealth(
			status,
			raw.Health,
			raw.ContextWindowUsagePct,
			raw.IdleSeconds,
			raw.StuckThresholdSeconds,
		)

		sessions = append(sessions, Session{
			ID:                    sessionID,
			Status:                nonEmpty(status, "running"),
			Provider:              strings.TrimSpace(raw.Provider),
			Model:                 strings.TrimSpace(raw.Model),
			TotalCostUSD:          raw.TotalCostUSD,
			DurationSeconds:       raw.DurationSeconds,
			LastActivity:          lastActivity,
			CurrentAction:         strings.TrimSpace(raw.CurrentAction),
			Health:                health,
			ContextWindowUsagePct: raw.ContextWindowUsagePct,
			RetryCount:            raw.RetryCount,
			IdleSeconds:           raw.IdleSeconds,
			StuckThresholdSeconds: raw.StuckThresholdSeconds,
			LoopState:             strings.TrimSpace(raw.LoopState),
		})
	}

	sort.SliceStable(sessions, func(i, j int) bool {
		if sessions[i].LastActivity.Equal(sessions[j].LastActivity) {
			return sessions[i].ID < sessions[j].ID
		}
		return sessions[i].LastActivity.After(sessions[j].LastActivity)
	})
	return sessions, nil
}

func readQueue(path string) ([]QueueItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read queue.json: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return nil, nil
	}

	var wrapped struct {
		Items []QueueItem `json:"items"`
	}
	if err := json.Unmarshal(data, &wrapped); err == nil {
		return wrapped.Items, nil
	}

	var items []QueueItem
	if err := json.Unmarshal(data, &items); err == nil {
		return items, nil
	}
	return nil, fmt.Errorf("parse queue.json")
}

func mapEventLines(events []event.Event) []EventLine {
	lines := make([]EventLine, 0, len(events))
	for _, ev := range events {
		line := EventLine{
			At:       ev.Timestamp,
			Label:    "Event",
			Body:     "",
			Category: traceFilterAll,
		}
		switch ev.Type {
		case event.EventCost:
			line.Label = "Cost"
			line.Category = traceFilterAll
			line.Body = formatCost(ev.Payload)
		case event.EventTicketClaim, event.EventTicketProgress, event.EventTicketDone, event.EventTicketBlocked, event.EventTicketRelease:
			line.Label = "Ticket"
			line.Category = traceFilterTicket
			line.Body = formatTicket(ev.Payload, string(ev.Type))
		case event.EventAction:
			label, body, category := formatAction(ev.Payload)
			line.Label = label
			line.Body = body
			line.Category = category
		default:
			line.Label = strings.Title(strings.ReplaceAll(string(ev.Type), "_", " "))
			line.Body = summarizePayload(ev.Payload)
			line.Category = traceFilterAll
		}
		if line.Body == "" {
			line.Body = "(no details)"
		}
		lines = append(lines, line)
	}
	return lines
}

func formatCost(payload json.RawMessage) string {
	var body struct {
		CostUSD   float64 `json:"cost_usd"`
		TokensIn  int     `json:"tokens_in"`
		TokensOut int     `json:"tokens_out"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return summarizePayload(payload)
	}
	if body.TokensIn == 0 && body.TokensOut == 0 {
		return fmt.Sprintf("$%.2f", body.CostUSD)
	}
	return fmt.Sprintf("$%.2f | %s in / %s out", body.CostUSD, shortInt(body.TokensIn), shortInt(body.TokensOut))
}

func formatTicket(payload json.RawMessage, fallback string) string {
	var body struct {
		Target  string `json:"target"`
		Summary string `json:"summary"`
		Outcome string `json:"outcome"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return fallback
	}
	if body.Summary != "" {
		return body.Summary
	}
	if body.Outcome != "" {
		return body.Outcome
	}
	if body.Reason != "" {
		return body.Reason
	}
	if body.Target != "" {
		return body.Target
	}
	return fallback
}

func formatAction(payload json.RawMessage) (label string, body string, category TraceFilter) {
	var action struct {
		Tool    string `json:"tool"`
		Action  string `json:"action"`
		Summary string `json:"summary"`
		Message string `json:"message"`
		Command string `json:"command"`
		Path    string `json:"path"`
	}
	_ = json.Unmarshal(payload, &action)

	tool := strings.ToLower(strings.TrimSpace(nonEmpty(action.Tool, action.Action)))
	switch tool {
	case "read":
		label = "Read"
		category = traceFilterTools
	case "edit", "write":
		label = "Edit"
		category = traceFilterTools
	case "bash", "shell":
		label = "Bash"
		category = traceFilterTools
	case "glob":
		label = "Glob"
		category = traceFilterTools
	case "grep":
		label = "Grep"
		category = traceFilterTools
	case "think":
		label = "Think"
		category = traceFilterThink
	default:
		// If no tool was recorded, this is usually assistant narration.
		label = "Think"
		category = traceFilterThink
	}

	body = firstNonEmpty(
		strings.TrimSpace(action.Summary),
		strings.TrimSpace(action.Message),
		strings.TrimSpace(action.Command),
		strings.TrimSpace(action.Path),
	)
	if body == "" {
		body = summarizePayload(payload)
	}
	return label, body, category
}

func summarizePayload(payload json.RawMessage) string {
	value := strings.TrimSpace(string(payload))
	if value == "" || value == "null" {
		return ""
	}
	return trimTo(value, 80)
}

func deriveHealth(status, explicit string, contextUsage float64, idleSeconds, stuckThresholdSeconds int64) string {
	explicit = strings.ToLower(strings.TrimSpace(explicit))
	if explicit == "green" || explicit == "yellow" || explicit == "red" {
		return explicit
	}

	status = strings.ToLower(strings.TrimSpace(status))
	if status == "failed" || status == "stuck" {
		return "red"
	}
	if status == "exited" || status == "killed" || status == "completed" {
		return "yellow"
	}

	if stuckThresholdSeconds <= 0 {
		stuckThresholdSeconds = defaultStuckThreshold
	}
	if idleSeconds > stuckThresholdSeconds {
		return "red"
	}
	if contextUsage >= 80 {
		return "yellow"
	}
	if idleSeconds > stuckThresholdSeconds/2 {
		return "yellow"
	}
	return "green"
}

func healthDot(health string) string {
	dot := "●"
	switch strings.ToLower(strings.TrimSpace(health)) {
	case "red":
		return errorStyle.Render(dot)
	case "yellow":
		return warnStyle.Render(dot)
	default:
		return successStyle.Render(dot)
	}
}

func statusIcon(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed":
		return errorStyle.Render("x")
	case "completed", "exited":
		return successStyle.Render("ok")
	default:
		return dimStyle.Render("~")
	}
}

func modelLabel(session Session) string {
	switch {
	case session.Provider != "" && session.Model != "":
		return session.Provider + "/" + session.Model
	case session.Model != "":
		return session.Model
	case session.Provider != "":
		return session.Provider
	default:
		return "-"
	}
}

func shortInt(v int) string {
	if v >= 1000 {
		return fmt.Sprintf("%.1fk", float64(v)/1000.0)
	}
	return fmt.Sprintf("%d", v)
}

func pickLoopState(current, candidate string) string {
	current = normalizeLoopState(current)
	candidate = normalizeLoopState(candidate)
	if candidate == "" {
		return current
	}
	if loopStateRank(candidate) > loopStateRank(current) {
		return candidate
	}
	return current
}

func normalizeLoopState(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "running":
		return "running"
	case "paused":
		return "paused"
	case "draining":
		return "draining"
	default:
		return ""
	}
}

func loopStateRank(state string) int {
	switch state {
	case "draining":
		return 3
	case "paused":
		return 2
	case "running":
		return 1
	default:
		return 0
	}
}

func isActiveStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running", "spawning", "stuck":
		return true
	default:
		return false
	}
}

func ageLabel(now, at time.Time) string {
	if at.IsZero() {
		return "-"
	}
	if now.Before(at) {
		return "0s"
	}
	seconds := int64(now.Sub(at).Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	return fmt.Sprintf("%dh", hours)
}

func durationLabel(seconds int64) string {
	if seconds <= 0 {
		return "0s"
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func trimTo(value string, width int) string {
	value = strings.TrimSpace(value)
	if width <= 0 || len(value) <= width {
		return value
	}
	if width <= 1 {
		return value[:width]
	}
	return value[:width-1] + "..."
}

func padRight(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) >= width {
		return string(runes[:width])
	}
	return value + strings.Repeat(" ", width-len(runes))
}

func wrapPlainText(value string, width int) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{""}
	}
	if width <= 1 {
		return []string{value}
	}
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	paragraphs := strings.Split(value, "\n")
	out := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			out = append(out, "")
			continue
		}
		out = append(out, wrapParagraph(paragraph, width)...)
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

func wrapParagraph(paragraph string, width int) []string {
	words := strings.Fields(paragraph)
	if len(words) == 0 {
		return []string{""}
	}
	lines := make([]string, 0, len(words))
	current := ""
	for _, word := range words {
		for runeLen(word) > width {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			head, tail := splitAtRuneWidth(word, width)
			lines = append(lines, head)
			word = tail
		}
		if current == "" {
			current = word
			continue
		}
		candidate := current + " " + word
		if runeLen(candidate) <= width {
			current = candidate
			continue
		}
		lines = append(lines, current)
		current = word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func runeLen(value string) int {
	return len([]rune(value))
}

func splitAtRuneWidth(value string, width int) (head string, tail string) {
	if width <= 0 {
		return "", value
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value, ""
	}
	return string(runes[:width]), string(runes[width:])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func parseTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	timestamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return timestamp.UTC()
}

func nonEmpty(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

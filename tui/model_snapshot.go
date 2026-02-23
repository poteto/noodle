package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/internal/queuex"
	"github.com/poteto/noodle/internal/sessionmeta"
	"github.com/poteto/noodle/internal/stringx"
	"github.com/poteto/noodle/loop"
)

func loadSnapshot(runtimeDir string, now time.Time) (Snapshot, error) {
	sessions, err := readSessions(runtimeDir)
	if err != nil {
		return Snapshot{}, err
	}
	qr, err := readQueue(filepath.Join(runtimeDir, "queue.json"))
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

	feedEvents := buildFeedEvents(sessions, eventsBySession)
	steerEvents := readSteerEvents(filepath.Join(runtimeDir, "control.ndjson"))
	feedEvents = append(feedEvents, steerEvents...)
	sort.SliceStable(feedEvents, func(i, j int) bool {
		return feedEvents[i].At.Before(feedEvents[j].At)
	})
	const maxFeedEvents = 500
	if len(feedEvents) > maxFeedEvents {
		feedEvents = feedEvents[len(feedEvents)-maxFeedEvents:]
	}

	brainDir := filepath.Join(filepath.Dir(runtimeDir), "brain")
	brainActivity := scanBrainActivity(brainDir)

	verdicts := loadVerdicts(runtimeDir)
	pendingCount := 0
	for _, v := range verdicts {
		if v.Accept {
			pendingCount++
		}
	}

	autonomy := readAutonomy(filepath.Dir(runtimeDir))

	return Snapshot{
		UpdatedAt:       now.UTC(),
		LoopState:       loopState,
		Sessions:        sessions,
		Active:          active,
		Recent:          recent,
		Queue:           qr.Items,
		ActiveQueueIDs:  qr.Active,
		ActionNeeded:    qr.ActionNeeded,
		EventsBySession: eventsBySession,
		FeedEvents:      feedEvents,
		TotalCostUSD:    totalCost,
		BrainActivity:   brainActivity,
		Verdicts:           verdicts,
		PendingReviewCount: pendingCount,
		Autonomy:           autonomy,
	}, nil
}

func readSessions(runtimeDir string) ([]Session, error) {
	metas, err := sessionmeta.ReadAll(runtimeDir)
	if err != nil {
		return nil, err
	}

	sessions := make([]Session, 0, len(metas))
	for _, meta := range metas {
		sessionID := strings.TrimSpace(meta.SessionID)
		status := strings.ToLower(strings.TrimSpace(meta.Status))
		health := deriveHealth(
			status,
			meta.Health,
			meta.ContextWindowUsagePct,
			meta.IdleSeconds,
			meta.StuckThresholdSeconds,
		)

		sessions = append(sessions, Session{
			ID:                    sessionID,
			DisplayName:           stringx.KitchenName(sessionID),
			Status:                stringx.NonEmpty(status, "running"),
			Provider:              strings.TrimSpace(meta.Provider),
			Model:                 strings.TrimSpace(meta.Model),
			TotalCostUSD:          meta.TotalCostUSD,
			DurationSeconds:       meta.DurationSeconds,
			LastActivity:          meta.LastActivity,
			CurrentAction:         strings.TrimSpace(meta.CurrentAction),
			Health:                health,
			ContextWindowUsagePct: meta.ContextWindowUsagePct,
			RetryCount:            meta.RetryCount,
			IdleSeconds:           meta.IdleSeconds,
			StuckThresholdSeconds: meta.StuckThresholdSeconds,
			LoopState:             strings.TrimSpace(meta.LoopState),
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

type queueResult struct {
	Items        []QueueItem
	Active       []string
	ActionNeeded []string
}

func readQueue(path string) (queueResult, error) {
	queue, err := queuex.Read(path)
	if err != nil {
		return queueResult{}, err
	}
	items := make([]QueueItem, 0, len(queue.Items))
	for _, item := range queue.Items {
		items = append(items, QueueItem{
			ID:        item.ID,
			TaskKey:   item.TaskKey,
			Title:     item.Title,
			Provider:  item.Provider,
			Model:     item.Model,
			Skill:     item.Skill,
			Review:    item.Review,
			Rationale: item.Rationale,
		})
	}
	return queueResult{
		Items:        items,
		Active:       queue.Active,
		ActionNeeded: queue.ActionNeeded,
	}, nil
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

	tool := strings.ToLower(strings.TrimSpace(stringx.NonEmpty(action.Tool, action.Action)))
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
	case "prompt":
		label = "Prompt"
		category = traceFilterThink
	case "think":
		label = "Think"
		category = traceFilterThink
	default:
		// If no tool was recorded, this is usually assistant narration.
		label = "Think"
		category = traceFilterThink
	}

	body = stringx.FirstNonEmpty(
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
	return stringx.MiddleTruncate(strings.TrimSpace(value), width)
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
	return stringx.NonEmpty(value, fallback)
}

// buildFeedEvents converts per-session EventLines into a flat feed timeline.
func buildFeedEvents(sessions []Session, eventsBySession map[string][]EventLine) []FeedEvent {
	feed := make([]FeedEvent, 0, 64)
	for _, session := range sessions {
		lines, ok := eventsBySession[session.ID]
		if !ok {
			continue
		}
		agentName := stringx.KitchenName(session.ID)
		taskType := inferTaskType(session.ID)
		for _, line := range lines {
			feed = append(feed, FeedEvent{
				SessionID: session.ID,
				AgentName: agentName,
				TaskType:  taskType,
				At:        line.At,
				Label:     line.Label,
				Body:      line.Body,
				Category:  string(line.Category),
			})
		}
	}
	return feed
}

// readSteerEvents reads control.ndjson and extracts steer commands as feed events.
func readSteerEvents(controlPath string) []FeedEvent {
	file, err := os.Open(controlPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var events []FeedEvent
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var cmd loop.ControlCommand
		if err := json.Unmarshal([]byte(line), &cmd); err != nil {
			continue
		}
		if !strings.EqualFold(cmd.Action, "steer") {
			continue
		}
		target := strings.TrimSpace(cmd.Target)
		prompt := strings.TrimSpace(cmd.Prompt)
		if target == "" && prompt == "" {
			continue
		}
		at := cmd.At
		if at.IsZero() {
			at = time.Now().UTC()
		}
		events = append(events, FeedEvent{
			SessionID: "chef",
			AgentName: target,
			At:        at,
			Label:     "Steer",
			Body:      prompt,
			Category:  "steer",
		})
	}
	return events
}

// inferTaskType extracts a task type from a session ID convention.
func inferTaskType(sessionID string) string {
	known := []string{"execute", "plan", "quality", "reflect", "prioritize"}
	lower := strings.ToLower(sessionID)
	for _, prefix := range known {
		if strings.HasPrefix(lower, prefix) {
			return prefix
		}
	}
	return ""
}

const brainScanLimit = 100

// scanBrainActivity walks the brain directory and returns recently modified
// markdown files sorted by mtime descending, capped to brainScanLimit.
func scanBrainActivity(brainDir string) []BrainActivity {
	info, err := os.Stat(brainDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	type entry struct {
		path  string
		mtime time.Time
	}
	var entries []entry

	_ = filepath.Walk(brainDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if fi.IsDir() {
			return nil
		}
		if !strings.HasSuffix(fi.Name(), ".md") {
			return nil
		}
		rel, err := filepath.Rel(brainDir, path)
		if err != nil {
			return nil
		}
		entries = append(entries, entry{path: rel, mtime: fi.ModTime()})
		return nil
	})

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].mtime.After(entries[j].mtime)
	})

	if len(entries) > brainScanLimit {
		entries = entries[:brainScanLimit]
	}

	activities := make([]BrainActivity, 0, len(entries))
	for _, e := range entries {
		tag := inferBrainTag(e.mtime)
		desc := inferDescription(e.path)
		activities = append(activities, BrainActivity{
			Agent:       "unknown",
			At:          e.mtime,
			Tag:         tag,
			FilePath:    e.path,
			Description: desc,
		})
	}
	return activities
}

func inferBrainTag(mtime time.Time) string {
	if time.Since(mtime) < time.Hour {
		return "new"
	}
	return "edit"
}

func inferDescription(relPath string) string {
	base := strings.TrimSuffix(filepath.Base(relPath), ".md")
	base = strings.ReplaceAll(base, "-", " ")
	return base
}

// readAutonomy reads the current autonomy mode from .noodle.toml.
// Returns "review" as default if config cannot be read.
func readAutonomy(projectDir string) string {
	configPath := filepath.Join(projectDir, config.DefaultConfigPath)
	cfg, _, err := config.Load(configPath)
	if err != nil {
		return config.AutonomyReview
	}
	if cfg.Autonomy == "" {
		return config.AutonomyReview
	}
	return cfg.Autonomy
}

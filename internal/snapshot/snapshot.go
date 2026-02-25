package snapshot

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
	"github.com/poteto/noodle/internal/statusfile"
	"github.com/poteto/noodle/internal/stringx"
	"github.com/poteto/noodle/loop"
)

const defaultStuckThreshold = int64(120)

// LoadSnapshot reads runtime state from disk and assembles a Snapshot.
func LoadSnapshot(runtimeDir string, now time.Time) (Snapshot, error) {
	sessions, err := readSessions(runtimeDir)
	if err != nil {
		return Snapshot{}, err
	}
	qr, err := readQueue(filepath.Join(runtimeDir, "queue.json"))
	if err != nil {
		return Snapshot{}, err
	}
	sr, err := readStatus(filepath.Join(runtimeDir, "status.json"))
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
	loopState := normalizeLoopState(sr.LoopState)
	if loopState == "" {
		loopState = "running"
	}
	for _, session := range sessions {
		totalCost += session.TotalCostUSD
		loopState = pickLoopState(loopState, session.LoopState)
		if IsActiveStatus(session.Status) {
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
	queueEvents := readQueueEvents(runtimeDir)
	feedEvents = append(feedEvents, queueEvents...)
	sort.SliceStable(feedEvents, func(i, j int) bool {
		return feedEvents[i].At.Before(feedEvents[j].At)
	})
	const maxFeedEvents = 500
	if len(feedEvents) > maxFeedEvents {
		feedEvents = feedEvents[len(feedEvents)-maxFeedEvents:]
	}

	pendingReviews, err := loop.ReadPendingReview(runtimeDir)
	if err != nil {
		return Snapshot{}, err
	}
	pendingCount := len(pendingReviews)

	autonomy := sr.Autonomy
	if autonomy == "" {
		autonomy = config.AutonomyAuto
	}

	return Snapshot{
		UpdatedAt:          now.UTC(),
		LoopState:          loopState,
		Sessions:           sessions,
		Active:             active,
		Recent:             recent,
		Queue:              qr.Items,
		ActiveQueueIDs:     sr.Active,
		ActionNeeded:       qr.ActionNeeded,
		EventsBySession:    eventsBySession,
		FeedEvents:         feedEvents,
		TotalCostUSD:       totalCost,
		PendingReviews:     pendingReviews,
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
			Runtime:               strings.TrimSpace(meta.Runtime),
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
			Prompt:    item.Prompt,
			Provider:  item.Provider,
			Model:     item.Model,
			Skill:     item.Skill,
			Plan:      item.Plan,
			Rationale: item.Rationale,
		})
	}
	return queueResult{
		Items:        items,
		ActionNeeded: queue.ActionNeeded,
	}, nil
}

type statusResult struct {
	Active    []string
	Autonomy  string
	LoopState string
}

func readStatus(path string) (statusResult, error) {
	status, err := statusfile.Read(path)
	if err != nil {
		return statusResult{}, err
	}
	return statusResult{
		Active:    status.Active,
		Autonomy:  status.Autonomy,
		LoopState: status.LoopState,
	}, nil
}

func mapEventLines(events []event.Event) []EventLine {
	lines := make([]EventLine, 0, len(events))
	for _, ev := range events {
		line := EventLine{
			At:       ev.Timestamp,
			Label:    "Event",
			Body:     "",
			Category: TraceFilterAll,
		}
		switch ev.Type {
		case event.EventCost:
			line.Label = "Cost"
			line.Category = TraceFilterAll
			line.Body = formatCost(ev.Payload)
		case event.EventTicketClaim, event.EventTicketProgress, event.EventTicketDone, event.EventTicketBlocked, event.EventTicketRelease:
			line.Label = "Ticket"
			line.Category = TraceFilterTicket
			line.Body = formatTicket(ev.Payload, string(ev.Type))
		case event.EventAction:
			label, body, category := formatAction(ev.Payload)
			line.Label = label
			line.Body = body
			line.Category = category
		case event.EventStateChange:
			line.Label = "State"
			line.Category = TraceFilterAll
			line.Body = formatStateChange(ev.Payload)
		default:
			line.Label = strings.Title(strings.ReplaceAll(string(ev.Type), "_", " "))
			line.Body = summarizePayload(ev.Payload)
			line.Category = TraceFilterAll
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
		category = TraceFilterTools
	case "edit", "write":
		label = "Edit"
		category = TraceFilterTools
	case "bash", "shell":
		label = "Bash"
		category = TraceFilterTools
	case "glob":
		label = "Glob"
		category = TraceFilterTools
	case "grep":
		label = "Grep"
		category = TraceFilterTools
	case "skill":
		label = "Skill"
		category = TraceFilterTools
	case "task":
		label = "Task"
		category = TraceFilterTools
	case "prompt":
		label = "Prompt"
		category = TraceFilterThink
	case "think":
		label = "Think"
		category = TraceFilterThink
	default:
		label = "Think"
		category = TraceFilterThink
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

func formatStateChange(payload json.RawMessage) string {
	var body struct {
		ToStatus string `json:"to_status"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return summarizePayload(payload)
	}
	status := strings.TrimSpace(body.ToStatus)
	reason := strings.TrimSpace(body.Reason)
	if status != "" && reason != "" {
		return status + ": " + reason
	}
	if reason != "" {
		return reason
	}
	if status != "" {
		return status
	}
	return summarizePayload(payload)
}

func summarizePayload(payload json.RawMessage) string {
	value := strings.TrimSpace(string(payload))
	if value == "" || value == "null" {
		return ""
	}
	return stringx.MiddleTruncate(value, 80)
}

func shortInt(v int) string {
	if v >= 1000 {
		return fmt.Sprintf("%.1fk", float64(v)/1000.0)
	}
	return fmt.Sprintf("%d", v)
}

func deriveHealth(status, explicit string, contextUsage float64, idleSeconds, stuckThresholdSeconds int64) string {
	explicit = strings.ToLower(strings.TrimSpace(explicit))
	if explicit == HealthGreen || explicit == HealthYellow || explicit == HealthRed {
		return explicit
	}

	status = strings.ToLower(strings.TrimSpace(status))
	if status == "failed" || status == "stuck" {
		return HealthRed
	}
	if status == "exited" || status == "killed" || status == "completed" {
		return HealthYellow
	}

	if stuckThresholdSeconds <= 0 {
		stuckThresholdSeconds = defaultStuckThreshold
	}
	if idleSeconds > stuckThresholdSeconds {
		return HealthRed
	}
	if contextUsage >= 80 {
		return HealthYellow
	}
	if idleSeconds > stuckThresholdSeconds/2 {
		return HealthYellow
	}
	return HealthGreen
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
	case "idle":
		return LoopStateIdle
	case "running":
		return LoopStateRunning
	case "paused":
		return LoopStatePaused
	case "draining":
		return LoopStateDraining
	default:
		return ""
	}
}

func loopStateRank(state string) int {
	switch state {
	case LoopStateDraining:
		return 3
	case LoopStatePaused:
		return 2
	case LoopStateRunning:
		return 1
	case LoopStateIdle:
		return 0
	default:
		return -1
	}
}

// IsActiveStatus returns true for statuses that indicate the session is active.
func IsActiveStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running", "spawning", "stuck":
		return true
	default:
		return false
	}
}

func buildFeedEvents(sessions []Session, eventsBySession map[string][]EventLine) []FeedEvent {
	feed := make([]FeedEvent, 0, 64)
	for _, session := range sessions {
		lines, ok := eventsBySession[session.ID]
		if !ok {
			continue
		}
		agentName := stringx.KitchenName(session.ID)
		taskType := InferTaskType(session.ID)
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

func readQueueEvents(runtimeDir string) []FeedEvent {
	path := filepath.Join(runtimeDir, "queue-events.ndjson")
	file, err := os.Open(path)
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
		var raw struct {
			At      time.Time `json:"at"`
			Type    string    `json:"type"`
			Target  string    `json:"target"`
			Skill   string    `json:"skill"`
			Reason  string    `json:"reason"`
			Added   []string  `json:"added"`
			Removed []string  `json:"removed"`
		}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}

		var label, body, category string
		switch raw.Type {
		case "queue_drop":
			label = "Dropped"
			category = "queue_drop"
			if raw.Reason != "" {
				body = fmt.Sprintf("Dropped item %s: %s", raw.Target, raw.Reason)
			} else {
				body = fmt.Sprintf("Dropped item %s: skill %s no longer exists", raw.Target, raw.Skill)
			}
		case "registry_rebuild":
			label = "Rebuild"
			category = "registry_rebuild"
			body = fmt.Sprintf("Registry rebuilt — added: %v, removed: %v", raw.Added, raw.Removed)
		case "bootstrap":
			label = "Bootstrap"
			category = "bootstrap"
			body = "Creating schedule skill from workflow analysis"
		default:
			continue
		}

		at := raw.At
		if at.IsZero() {
			at = time.Now().UTC()
		}
		events = append(events, FeedEvent{
			SessionID: "loop",
			AgentName: "loop",
			At:        at,
			Label:     label,
			Body:      body,
			Category:  category,
		})
	}
	return events
}

// InferTaskType extracts a task type from a session ID convention.
func InferTaskType(sessionID string) string {
	known := []string{"execute", "plan", "review", "reflect", "schedule"}
	lower := strings.ToLower(sessionID)
	for _, prefix := range known {
		if strings.HasPrefix(lower, prefix) {
			return prefix
		}
	}
	return ""
}

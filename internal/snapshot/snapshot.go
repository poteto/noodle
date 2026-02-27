package snapshot

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/internal/stringx"
	"github.com/poteto/noodle/loop"
)

// LoadSnapshot builds a Snapshot from in-memory LoopState. Session data comes
// from the loop's active cook tracking and recent history — no directory
// scanning or bulk event reading.
func LoadSnapshot(runtimeDir string, now time.Time, state loop.LoopState) (Snapshot, error) {
	sessions := make([]Session, 0, len(state.ActiveCooks)+len(state.RecentHistory))
	active := make([]Session, 0, len(state.ActiveCooks))
	recent := make([]Session, 0, len(state.RecentHistory))

	for _, cook := range state.ActiveCooks {
		status := strings.ToLower(strings.TrimSpace(cook.Status))
		if status == "" {
			status = "running"
		}
		session := Session{
			ID:           cook.SessionID,
			DisplayName:  cook.DisplayName,
			Status:       status,
			Runtime:      cook.Runtime,
			Provider:     cook.Provider,
			Model:        cook.Model,
			LastActivity: now.UTC(),
			TaskKey:      cook.TaskKey,
		}
		sessions = append(sessions, session)
		active = append(active, session)
	}

	for _, item := range state.RecentHistory {
		session := Session{
			ID:           item.SessionID,
			DisplayName:  item.SessionID,
			Status:       strings.ToLower(strings.TrimSpace(item.Status)),
			LastActivity: item.CompletedAt,
			TaskKey:      item.TaskKey,
		}
		sessions = append(sessions, session)
		recent = append(recent, session)
	}

	sort.SliceStable(active, func(i, j int) bool {
		return active[i].ID < active[j].ID
	})
	sort.SliceStable(recent, func(i, j int) bool {
		if recent[i].LastActivity.Equal(recent[j].LastActivity) {
			return recent[i].ID < recent[j].ID
		}
		return recent[i].LastActivity.After(recent[j].LastActivity)
	})

	// Build a lookup from order ID to active cook session ID.
	cookSessionByOrder := make(map[string]string, len(state.ActiveCooks))
	for _, cook := range state.ActiveCooks {
		cookSessionByOrder[cook.OrderID] = cook.SessionID
	}

	orders := make([]Order, 0, len(state.Orders))
	for _, order := range state.Orders {
		activeSessionID := cookSessionByOrder[order.ID]
		stages := make([]Stage, 0, len(order.Stages))
		for _, stage := range order.Stages {
			s := Stage{
				TaskKey:  stage.TaskKey,
				Prompt:   stage.Prompt,
				Skill:    stage.Skill,
				Provider: stage.Provider,
				Model:    stage.Model,
				Runtime:  stage.Runtime,
				Status:   stage.Status,
				Extra:    stage.Extra,
			}
			if (stage.Status == loop.StageStatusActive || stage.Status == loop.StageStatusMerging) && activeSessionID != "" {
				s.SessionID = activeSessionID
			}
			stages = append(stages, s)
		}
		onFailure := make([]Stage, 0, len(order.OnFailure))
		for _, stage := range order.OnFailure {
			s := Stage{
				TaskKey:  stage.TaskKey,
				Prompt:   stage.Prompt,
				Skill:    stage.Skill,
				Provider: stage.Provider,
				Model:    stage.Model,
				Runtime:  stage.Runtime,
				Status:   stage.Status,
				Extra:    stage.Extra,
			}
			if (stage.Status == loop.StageStatusActive || stage.Status == loop.StageStatusMerging) && activeSessionID != "" {
				s.SessionID = activeSessionID
			}
			onFailure = append(onFailure, s)
		}
		orders = append(orders, Order{
			ID:        order.ID,
			Title:     order.Title,
			Plan:      append([]string(nil), order.Plan...),
			Rationale: order.Rationale,
			Stages:    stages,
			Status:    order.Status,
			OnFailure: onFailure,
		})
	}

	feedEvents := readSteerEvents(filepath.Join(runtimeDir, "control.ndjson"))
	queueEvents := readQueueEvents(runtimeDir)
	feedEvents = append(feedEvents, queueEvents...)
	sort.SliceStable(feedEvents, func(i, j int) bool {
		return feedEvents[i].At.Before(feedEvents[j].At)
	})

	return Snapshot{
		UpdatedAt:          now.UTC(),
		LoopState:          normalizeLoopState(state.Status),
		Sessions:           sessions,
		Active:             active,
		Recent:             recent,
		Orders:             orders,
		ActiveOrderIDs:     nonNilStrings(state.ActiveOrderIDs),
		ActionNeeded:       nonNilStrings(state.ActionNeeded),
		EventsBySession:    map[string][]EventLine{},
		FeedEvents:         nonNilFeedEvents(feedEvents),
		TotalCostUSD:       state.TotalCostUSD,
		PendingReviews:     nonNilReviews(state.PendingReviews),
		PendingReviewCount: state.PendingReviewCount,
		Autonomy:           state.Autonomy,
		MaxCooks:           state.MaxCooks,
	}, nil
}

// ReadSessionEvents reads canonical events for a single session on demand.
func ReadSessionEvents(runtimeDir, sessionID string) ([]EventLine, error) {
	reader := event.NewEventReader(runtimeDir)
	events, err := reader.ReadSession(sessionID, event.EventFilter{})
	if err != nil {
		if os.IsNotExist(err) {
			return []EventLine{}, nil
		}
		return nil, err
	}
	return mapEventLines(events), nil
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
		case "order_drop", "queue_drop":
			label = "Dropped"
			category = "order_drop"
			if raw.Reason != "" {
				body = fmt.Sprintf("Dropped order %s: %s", raw.Target, raw.Reason)
			} else {
				body = fmt.Sprintf("Dropped order %s: skill %s no longer exists", raw.Target, raw.Skill)
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

// nonNil helpers ensure slices marshal to [] instead of null in JSON.

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return append([]string(nil), s...)
}

func nonNilFeedEvents(s []FeedEvent) []FeedEvent {
	if s == nil {
		return []FeedEvent{}
	}
	return s
}

func nonNilReviews(s []loop.PendingReviewItem) []loop.PendingReviewItem {
	if s == nil {
		return []loop.PendingReviewItem{}
	}
	return append([]loop.PendingReviewItem(nil), s...)
}

// InferTaskType extracts a task type from session/worktree naming conventions.
// Supported formats:
// - orderID:stageIndex:taskKey (legacy)
// - order-id-stageIndex-task-key (current, dasherized)
func InferTaskType(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return ""
	}

	// Legacy colon-separated format: orderID:stageIndex:taskKey
	parts := strings.SplitN(lower, ":", 3)
	if len(parts) == 3 {
		taskKey := strings.TrimSpace(parts[2])
		if idx := strings.IndexByte(taskKey, '-'); idx > 0 {
			taskKey = taskKey[:idx]
		}
		if taskKey != "" {
			return taskKey
		}
	}

	known := []string{"execute", "plan", "review", "reflect", "schedule", "quality", "debugging", "meditate", "oops", "bootstrap", "repair", "request-changes"}

	// Dasherized format: find the stage index segment and read the trailing task key.
	if dashParts := strings.Split(lower, "-"); len(dashParts) >= 3 {
		for i := len(dashParts) - 2; i >= 0; i-- {
			if _, err := strconv.Atoi(dashParts[i]); err != nil {
				continue
			}
			taskCandidate := strings.Join(dashParts[i+1:], "-")
			for _, key := range known {
				if taskCandidate == key || strings.HasPrefix(taskCandidate, key+"-") {
					return key
				}
			}
		}
	}

	// Older prefix-based session IDs (e.g., "execute-...").
	for _, prefix := range known {
		if strings.HasPrefix(lower, prefix) {
			return prefix
		}
	}
	return ""
}

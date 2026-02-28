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
	"github.com/poteto/noodle/internal/orderx"
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

	reader := event.NewEventReader(runtimeDir)
	for _, cook := range state.ActiveCooks {
		status := strings.ToLower(strings.TrimSpace(cook.Status))
		if status == "" {
			status = "running"
		}
		session := Session{
			ID:              cook.SessionID,
			DisplayName:     cook.DisplayName,
			Status:          status,
			Runtime:         cook.Runtime,
			Provider:        cook.Provider,
			Model:           cook.Model,
			LastActivity:    now.UTC(),
			TaskKey:         cook.TaskKey,
			TotalCostUSD:    cook.TotalCostUSD,
			DurationSeconds: int64(now.Sub(cook.StartedAt).Seconds()),
		}
		enrichActiveSession(reader, cook.SessionID, &session)
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
				Status:   string(stage.Status),
				Extra:    stage.Extra,
			}
			if (stage.Status == orderx.StageStatusActive || stage.Status == orderx.StageStatusMerging) && activeSessionID != "" {
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
				Status:   string(stage.Status),
				Extra:    stage.Extra,
			}
			if (stage.Status == orderx.StageStatusActive || stage.Status == orderx.StageStatusMerging) && activeSessionID != "" {
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
			Status:    string(order.Status),
			OnFailure: onFailure,
		})
	}

	feedEvents := readSteerEvents(filepath.Join(runtimeDir, "control.ndjson"))
	loopEvents := readLoopEvents(runtimeDir)
	feedEvents = append(feedEvents, loopEvents...)
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
		ActiveOrderIDs:     nonNilStrings(cookingOrderIDs(cookSessionByOrder)),
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

// enrichActiveSession reads the session's event log and populates
// CurrentAction and ContextWindowUsagePct on the given Session.
func enrichActiveSession(reader *event.EventReader, sessionID string, session *Session) {
	events, err := reader.ReadSession(sessionID, event.EventFilter{})
	if err != nil || len(events) == 0 {
		return
	}

	var totalTokensIn int
	for _, ev := range events {
		switch ev.Type {
		case event.EventAction:
			label, body, _ := formatAction(ev.Payload)
			if body != "" {
				session.CurrentAction = label + " " + body
			}
		case event.EventCost:
			var cost struct {
				TokensIn int `json:"tokens_in"`
			}
			if json.Unmarshal(ev.Payload, &cost) == nil {
				totalTokensIn += cost.TokensIn
			}
		}
	}

	pct := float64(totalTokensIn) / 200_000 * 100
	if pct > 100 {
		pct = 100
	}
	session.ContextWindowUsagePct = pct
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

func readLoopEvents(runtimeDir string) []FeedEvent {
	path := filepath.Join(runtimeDir, "loop-events.ndjson")
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
			Seq     uint64          `json:"seq"`
			Type    string          `json:"type"`
			At      time.Time       `json:"at"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}

		var label, body, category, taskType string
		switch raw.Type {
		case "stage.completed":
			label = "Completed"
			category = "stage_completed"
			var p struct {
				OrderID string  `json:"order_id"`
				TaskKey string  `json:"task_key"`
				Message *string `json:"message"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			taskType = p.TaskKey
			if p.Message != nil && *p.Message != "" {
				body = *p.Message
			}
		case "stage.failed":
			label = "Failed"
			category = "stage_failed"
			var p struct {
				OrderID string `json:"order_id"`
				TaskKey string `json:"task_key"`
				Reason  string `json:"reason"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			taskType = p.TaskKey
			body = p.Reason
		case "order.completed":
			label = "Order Complete"
			category = "order_completed"
			var p struct {
				OrderID string `json:"order_id"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			body = p.OrderID
		case "order.failed":
			label = "Order Failed"
			category = "order_failed"
			var p struct {
				OrderID string `json:"order_id"`
				Reason  string `json:"reason"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			if p.Reason != "" {
				body = p.Reason
			} else {
				body = p.OrderID
			}
		case "order.dropped":
			label = "Dropped"
			category = "order_drop"
			var p struct {
				OrderID string `json:"order_id"`
				Reason  string `json:"reason"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			if p.Reason != "" {
				body = fmt.Sprintf("Dropped order %s: %s", p.OrderID, p.Reason)
			} else {
				body = fmt.Sprintf("Dropped order %s", p.OrderID)
			}
		case "order.requeued":
			label = "Requeued"
			category = "order_requeued"
			var p struct {
				OrderID string `json:"order_id"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			body = p.OrderID
		case "worktree.merged":
			label = "Merged"
			category = "worktree_merged"
			var p struct {
				OrderID      string `json:"order_id"`
				WorktreeName string `json:"worktree_name"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			body = p.WorktreeName
		case "merge.conflict":
			label = "Conflict"
			category = "merge_conflict"
			var p struct {
				OrderID      string `json:"order_id"`
				WorktreeName string `json:"worktree_name"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			body = p.WorktreeName
		case "schedule.completed":
			label = "Scheduled"
			category = "schedule_completed"
			taskType = "schedule"
		case "registry.rebuilt":
			label = "Rebuild"
			category = "registry_rebuild"
			var p struct {
				Added   []string `json:"added"`
				Removed []string `json:"removed"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			body = fmt.Sprintf("Registry rebuilt — added: %v, removed: %v", p.Added, p.Removed)
		case "bootstrap.completed":
			label = "Bootstrap"
			category = "bootstrap"
			body = "Bootstrap completed"
		case "bootstrap.exhausted":
			label = "Bootstrap"
			category = "bootstrap"
			var p struct {
				Reason string `json:"reason"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			if p.Reason != "" {
				body = p.Reason
			} else {
				body = "Bootstrap exhausted"
			}
		case "sync.degraded":
			label = "Sync"
			category = "sync_degraded"
			var p struct {
				Reason string `json:"reason"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			if p.Reason != "" {
				body = p.Reason
			} else {
				body = "Sync degraded"
			}
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
			TaskType:  taskType,
			At:        at,
			Label:     label,
			Body:      body,
			Category:  category,
		})
	}
	return events
}

// cookingOrderIDs returns order IDs that have an active cook session.
// This is a subset of the loop's ActiveOrderIDs — it excludes orders that are
// "active" but waiting to be dispatched (no cook yet).
func cookingOrderIDs(sessionByOrder map[string]string) []string {
	ids := make([]string, 0, len(sessionByOrder))
	for orderID := range sessionByOrder {
		ids = append(ids, orderID)
	}
	sort.Strings(ids)
	return ids
}

// nonNil helpers ensure slices marshal to [] instead of null in JSON.

func nonNilStrings(s []string) []string {
	if len(s) == 0 {
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
// Format: order-id-stageIndex-task-key (dasherized)
func InferTaskType(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return ""
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

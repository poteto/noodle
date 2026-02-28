package reducer

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/state"
)

// Reduce is the default reducer implementation.
func Reduce(current state.State, event ingest.StateEvent) (state.State, []Effect) {
	switch ingest.EventType(event.Type) {
	case ingest.EventDispatchRequested:
		return reduceDispatchRequested(current, event)
	case ingest.EventDispatchCompleted:
		return reduceDispatchCompleted(current, event)
	case ingest.EventStageCompleted:
		return reduceStageCompleted(current, event)
	case ingest.EventStageFailed:
		return reduceStageFailed(current, event)
	case ingest.EventOrderCompleted:
		return reduceOrderCompleted(current, event)
	case ingest.EventOrderFailed:
		return reduceOrderFailed(current, event)
	case ingest.EventModeChanged:
		return reduceModeChanged(current, event)
	case ingest.EventMergeCompleted:
		return reduceMergeCompleted(current, event)
	case ingest.EventMergeFailed:
		return reduceMergeFailed(current, event)
	default:
		return current, nil
	}
}

// DefaultReducer returns the canonical reducer function.
func DefaultReducer() Reducer {
	return Reduce
}

type orderStagePayload struct {
	OrderID      string `json:"order_id"`
	StageIndex   int    `json:"stage_index"`
	AttemptID    string `json:"attempt_id"`
	SessionID    string `json:"session_id"`
	WorktreeName string `json:"worktree_name"`
	Error        string `json:"error"`
	Mergeable    *bool  `json:"mergeable"`
	ExitCode     *int   `json:"exit_code"`
}

type orderPayload struct {
	OrderID string `json:"order_id"`
}

type modePayload struct {
	Mode string `json:"mode"`
}

func reduceDispatchRequested(current state.State, event ingest.StateEvent) (state.State, []Effect) {
	var payload orderStagePayload
	if !decodeEventPayload(event, &payload) {
		return current, nil
	}
	order, stage, ok := lookupOrderStage(current, payload.OrderID, payload.StageIndex)
	if !ok || isTerminalOrder(order.Status) || isTerminalStage(stage.Status) {
		return current, nil
	}

	next := cloneState(current)
	order = next.Orders[payload.OrderID]
	stage = order.Stages[payload.StageIndex]
	stage.Status = state.StageDispatching
	order.Status = state.OrderActive
	order.UpdatedAt = event.Timestamp
	order.Stages[payload.StageIndex] = stage
	next.Orders[payload.OrderID] = order
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)

	effects := []Effect{
		makeEffect(event, 0, EffectDispatch, map[string]any{
			"order_id":    payload.OrderID,
			"stage_index": payload.StageIndex,
			"attempt_id":  normalizedAttemptID(payload.AttemptID, event),
		}),
	}
	return next, effects
}

func reduceDispatchCompleted(current state.State, event ingest.StateEvent) (state.State, []Effect) {
	var payload orderStagePayload
	if !decodeEventPayload(event, &payload) {
		return current, nil
	}
	order, stage, ok := lookupOrderStage(current, payload.OrderID, payload.StageIndex)
	if !ok || isTerminalOrder(order.Status) || isTerminalStage(stage.Status) {
		return current, nil
	}

	attemptID := normalizedAttemptID(payload.AttemptID, event)

	next := cloneState(current)
	order = next.Orders[payload.OrderID]
	stage = order.Stages[payload.StageIndex]
	stage.Status = state.StageRunning
	order.Status = state.OrderActive
	order.UpdatedAt = event.Timestamp

	attempt := state.AttemptNode{
		AttemptID:    attemptID,
		SessionID:    payload.SessionID,
		Status:       state.AttemptRunning,
		StartedAt:    event.Timestamp,
		WorktreeName: payload.WorktreeName,
	}
	if idx, found := attemptIndexByID(stage.Attempts, attemptID); found {
		existing := stage.Attempts[idx]
		existing.SessionID = attempt.SessionID
		existing.Status = attempt.Status
		existing.StartedAt = attempt.StartedAt
		existing.WorktreeName = attempt.WorktreeName
		stage.Attempts[idx] = existing
	} else {
		stage.Attempts = append(stage.Attempts, attempt)
	}

	order.Stages[payload.StageIndex] = stage
	next.Orders[payload.OrderID] = order
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)
	return next, nil
}

func reduceStageCompleted(current state.State, event ingest.StateEvent) (state.State, []Effect) {
	var payload orderStagePayload
	if !decodeEventPayload(event, &payload) {
		return current, nil
	}
	order, stage, ok := lookupOrderStage(current, payload.OrderID, payload.StageIndex)
	if !ok || isTerminalOrder(order.Status) || isTerminalStage(stage.Status) {
		return current, nil
	}

	next := cloneState(current)
	order = next.Orders[payload.OrderID]
	stage = order.Stages[payload.StageIndex]
	stage.Status = state.StageCompleted
	order.UpdatedAt = event.Timestamp
	finalizeAttempt(&stage, payload, event, state.AttemptCompleted)
	order.Stages[payload.StageIndex] = stage
	next.Orders[payload.OrderID] = order
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)

	if !stageMergeable(stage, payload) {
		return next, nil
	}

	effects := []Effect{
		makeEffect(event, 0, EffectMerge, map[string]any{
			"order_id":      payload.OrderID,
			"stage_index":   payload.StageIndex,
			"worktree_name": latestWorktreeName(stage),
		}),
	}
	return next, effects
}

func reduceStageFailed(current state.State, event ingest.StateEvent) (state.State, []Effect) {
	var payload orderStagePayload
	if !decodeEventPayload(event, &payload) {
		return current, nil
	}
	order, stage, ok := lookupOrderStage(current, payload.OrderID, payload.StageIndex)
	if !ok || isTerminalOrder(order.Status) || isTerminalStage(stage.Status) {
		return current, nil
	}

	next := cloneState(current)
	order = next.Orders[payload.OrderID]
	stage = order.Stages[payload.StageIndex]
	stage.Status = state.StageFailed
	order.UpdatedAt = event.Timestamp
	finalizeAttempt(&stage, payload, event, state.AttemptFailed)
	order.Stages[payload.StageIndex] = stage
	next.Orders[payload.OrderID] = order
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)

	effects := []Effect{
		makeEffect(event, 0, EffectCleanup, map[string]any{
			"order_id":      payload.OrderID,
			"stage_index":   payload.StageIndex,
			"worktree_name": latestWorktreeName(stage),
		}),
	}
	return next, effects
}

func reduceOrderCompleted(current state.State, event ingest.StateEvent) (state.State, []Effect) {
	var payload orderPayload
	if !decodeEventPayload(event, &payload) {
		return current, nil
	}
	order, ok := current.Orders[payload.OrderID]
	if !ok || isTerminalOrder(order.Status) {
		return current, nil
	}

	next := cloneState(current)
	order = next.Orders[payload.OrderID]
	order.Status = state.OrderCompleted
	order.UpdatedAt = event.Timestamp
	next.Orders[payload.OrderID] = order
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)

	effects := []Effect{
		makeEffect(event, 0, EffectWriteProjection, map[string]any{"order_id": payload.OrderID}),
		makeEffect(event, 1, EffectAck, map[string]any{"order_id": payload.OrderID}),
	}
	return next, effects
}

func reduceOrderFailed(current state.State, event ingest.StateEvent) (state.State, []Effect) {
	var payload orderPayload
	if !decodeEventPayload(event, &payload) {
		return current, nil
	}
	order, ok := current.Orders[payload.OrderID]
	if !ok || isTerminalOrder(order.Status) {
		return current, nil
	}

	next := cloneState(current)
	order = next.Orders[payload.OrderID]
	order.Status = state.OrderFailed
	order.UpdatedAt = event.Timestamp
	next.Orders[payload.OrderID] = order
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)

	effects := []Effect{
		makeEffect(event, 0, EffectWriteProjection, map[string]any{"order_id": payload.OrderID}),
		makeEffect(event, 1, EffectAck, map[string]any{"order_id": payload.OrderID}),
	}
	return next, effects
}

func reduceModeChanged(current state.State, event ingest.StateEvent) (state.State, []Effect) {
	var payload modePayload
	if !decodeEventPayload(event, &payload) {
		return current, nil
	}
	mode, ok := parseRunMode(payload.Mode)
	if !ok {
		return current, nil
	}

	next := cloneState(current)
	next.Mode = mode
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)
	return next, nil
}

func reduceMergeCompleted(current state.State, event ingest.StateEvent) (state.State, []Effect) {
	var payload orderStagePayload
	if !decodeEventPayload(event, &payload) {
		return current, nil
	}
	order, stage, ok := lookupOrderStage(current, payload.OrderID, payload.StageIndex)
	if !ok || isTerminalOrder(order.Status) {
		return current, nil
	}
	if stage.Status == state.StageFailed || stage.Status == state.StageCancelled || stage.Status == state.StageSkipped {
		return current, nil
	}

	next := cloneState(current)
	order = next.Orders[payload.OrderID]
	stage = order.Stages[payload.StageIndex]
	stage.Status = state.StageCompleted
	order.Stages[payload.StageIndex] = stage
	order.UpdatedAt = event.Timestamp

	nextStageIndex := payload.StageIndex + 1
	if nextStageIndex >= len(order.Stages) {
		order.Status = state.OrderCompleted
	} else {
		order.Status = state.OrderActive
		if order.Stages[nextStageIndex].Status == "" {
			order.Stages[nextStageIndex].Status = state.StagePending
		}
	}

	next.Orders[payload.OrderID] = order
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)
	return next, nil
}

func reduceMergeFailed(current state.State, event ingest.StateEvent) (state.State, []Effect) {
	var payload orderStagePayload
	if !decodeEventPayload(event, &payload) {
		return current, nil
	}
	order, stage, ok := lookupOrderStage(current, payload.OrderID, payload.StageIndex)
	if !ok || isTerminalOrder(order.Status) || isTerminalStage(stage.Status) {
		return current, nil
	}

	next := cloneState(current)
	order = next.Orders[payload.OrderID]
	stage = order.Stages[payload.StageIndex]
	stage.Status = state.StageReview
	order.UpdatedAt = event.Timestamp
	order.Stages[payload.StageIndex] = stage
	next.Orders[payload.OrderID] = order
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)

	effects := []Effect{
		makeEffect(event, 0, EffectAck, map[string]any{
			"order_id":    payload.OrderID,
			"stage_index": payload.StageIndex,
		}),
	}
	return next, effects
}

func decodeEventPayload(event ingest.StateEvent, out any) bool {
	if len(event.Payload) == 0 {
		return false
	}
	if err := json.Unmarshal(event.Payload, out); err != nil {
		return false
	}
	return true
}

func parseRunMode(raw string) (state.RunMode, bool) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch state.RunMode(normalized) {
	case state.RunModeAuto, state.RunModeSupervised, state.RunModeManual:
		return state.RunMode(normalized), true
	default:
		return "", false
	}
}

func lookupOrderStage(s state.State, orderID string, stageIndex int) (state.OrderNode, state.StageNode, bool) {
	if stageIndex < 0 {
		return state.OrderNode{}, state.StageNode{}, false
	}
	order, ok := s.Orders[orderID]
	if !ok {
		return state.OrderNode{}, state.StageNode{}, false
	}
	if stageIndex >= len(order.Stages) {
		return state.OrderNode{}, state.StageNode{}, false
	}
	return order, order.Stages[stageIndex], true
}

func normalizedAttemptID(raw string, event ingest.StateEvent) string {
	id := strings.TrimSpace(raw)
	if id != "" {
		return id
	}
	return fmt.Sprintf("attempt-%d", event.ID)
}

func attemptIndexByID(attempts []state.AttemptNode, attemptID string) (int, bool) {
	for i := range attempts {
		if attempts[i].AttemptID == attemptID {
			return i, true
		}
	}
	return -1, false
}

func finalizeAttempt(stage *state.StageNode, payload orderStagePayload, event ingest.StateEvent, status state.AttemptStatus) {
	if len(stage.Attempts) == 0 {
		return
	}

	attemptIndex := -1
	if trimmed := strings.TrimSpace(payload.AttemptID); trimmed != "" {
		if idx, found := attemptIndexByID(stage.Attempts, trimmed); found {
			attemptIndex = idx
		}
	}
	if attemptIndex < 0 {
		attemptIndex = len(stage.Attempts) - 1
	}

	attempt := stage.Attempts[attemptIndex]
	attempt.Status = status
	attempt.CompletedAt = event.Timestamp
	if payload.ExitCode != nil {
		exitCode := *payload.ExitCode
		attempt.ExitCode = &exitCode
	}
	if strings.TrimSpace(payload.Error) != "" {
		attempt.Error = strings.TrimSpace(payload.Error)
	}
	if strings.TrimSpace(payload.WorktreeName) != "" {
		attempt.WorktreeName = strings.TrimSpace(payload.WorktreeName)
	}
	stage.Attempts[attemptIndex] = attempt
}

func latestWorktreeName(stage state.StageNode) string {
	if len(stage.Attempts) == 0 {
		return ""
	}
	for i := len(stage.Attempts) - 1; i >= 0; i-- {
		if name := strings.TrimSpace(stage.Attempts[i].WorktreeName); name != "" {
			return name
		}
	}
	return ""
}

func stageMergeable(stage state.StageNode, payload orderStagePayload) bool {
	if payload.Mergeable != nil {
		return *payload.Mergeable
	}
	return latestWorktreeName(stage) != ""
}

func isTerminalOrder(status state.OrderLifecycleStatus) bool {
	switch status {
	case state.OrderCompleted, state.OrderFailed, state.OrderCancelled:
		return true
	default:
		return false
	}
}

func isTerminalStage(status state.StageLifecycleStatus) bool {
	switch status {
	case state.StageCompleted, state.StageFailed, state.StageSkipped, state.StageCancelled:
		return true
	default:
		return false
	}
}

func cloneState(in state.State) state.State {
	out := in
	if in.Orders == nil {
		out.Orders = nil
		return out
	}

	out.Orders = make(map[string]state.OrderNode, len(in.Orders))
	for orderID, order := range in.Orders {
		orderCopy := order

		if order.Metadata != nil {
			metadataCopy := make(map[string]string, len(order.Metadata))
			for key, value := range order.Metadata {
				metadataCopy[key] = value
			}
			orderCopy.Metadata = metadataCopy
		}

		if order.Stages != nil {
			stagesCopy := make([]state.StageNode, len(order.Stages))
			for i := range order.Stages {
				stageCopy := order.Stages[i]
				if order.Stages[i].Attempts != nil {
					attemptsCopy := make([]state.AttemptNode, len(order.Stages[i].Attempts))
					for j := range order.Stages[i].Attempts {
						attemptCopy := order.Stages[i].Attempts[j]
						if attemptCopy.ExitCode != nil {
							exitCode := *attemptCopy.ExitCode
							attemptCopy.ExitCode = &exitCode
						}
						attemptsCopy[j] = attemptCopy
					}
					stageCopy.Attempts = attemptsCopy
				}
				stagesCopy[i] = stageCopy
			}
			orderCopy.Stages = stagesCopy
		}

		out.Orders[orderID] = orderCopy
	}

	return out
}

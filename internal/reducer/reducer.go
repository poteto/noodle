package reducer

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/state"
	"github.com/poteto/noodle/internal/stringx"
)

type reduceHandler func(state.State, ingest.StateEvent) (state.State, []Effect, error)

var reduceHandlers = map[ingest.EventType]reduceHandler{
	ingest.EventDispatchRequested:           reduceDispatchRequested,
	ingest.EventDispatchCompleted:           reduceDispatchCompleted,
	ingest.EventStageCompleted:              reduceStageCompleted,
	ingest.EventStageFailed:                 reduceStageFailed,
	ingest.EventStageReviewParked:           reduceStageReviewParked,
	ingest.EventStageReviewApproved:         reduceStageReviewApproved,
	ingest.EventStageReviewChangesRequested: reduceStageReviewChangesRequested,
	ingest.EventStageReviewRejected:         reduceStageReviewRejected,
	ingest.EventOrderCompleted:              reduceOrderCompleted,
	ingest.EventOrderFailed:                 reduceOrderFailed,
	ingest.EventModeChanged:                 reduceModeChanged,
	ingest.EventMergeCompleted:              reduceMergeCompleted,
	ingest.EventMergeFailed:                 reduceMergeFailed,
	ingest.EventControlReceived:             reduceControlReceived,
	ingest.EventSchedulePromoted:            reduceSchedulePromoted,
	ingest.EventSessionAdopted:              reduceSessionAdopted,
}

// Reduce is the default reducer implementation.
func Reduce(current state.State, event ingest.StateEvent) (state.State, []Effect, error) {
	eventType := ingest.EventType(event.Type)
	if !ingest.IsKnownEventType(eventType) {
		return current, nil, nil
	}
	handler, ok := reduceHandlers[eventType]
	if !ok {
		return current, nil, fmt.Errorf("event type has no reducer handler: %q", event.Type)
	}
	return handler(current, event)
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
	MergeMode    string `json:"merge_mode"`
	MergeBranch  string `json:"merge_branch"`
	Error        string `json:"error"`
	Mergeable    *bool  `json:"mergeable"`
	ExitCode     *int   `json:"exit_code"`
}

type reviewPayload struct {
	OrderID      string   `json:"order_id"`
	StageIndex   int      `json:"stage_index"`
	AttemptID    string   `json:"attempt_id"`
	SessionID    string   `json:"session_id"`
	WorktreeName string   `json:"worktree_name"`
	WorktreePath string   `json:"worktree_path"`
	Reason       string   `json:"reason"`
	TaskKey      string   `json:"task_key"`
	Prompt       string   `json:"prompt"`
	Provider     string   `json:"provider"`
	Model        string   `json:"model"`
	Runtime      string   `json:"runtime"`
	Skill        string   `json:"skill"`
	Plan         []string `json:"plan"`
}

type orderPayload struct {
	OrderID string `json:"order_id"`
}

type modePayload struct {
	Mode        string `json:"mode"`
	RequestedBy string `json:"requested_by"`
	Reason      string `json:"reason"`
}

type controlPayload struct {
	Command string `json:"command"`
	Mode    string `json:"mode"`
}

type schedulePromotedPayload struct {
	OrderID  string            `json:"order_id"`
	Stages   []state.StageNode `json:"stages"`
	Metadata map[string]string `json:"metadata"`
}

type sessionAdoptedPayload struct {
	OrderID    string `json:"order_id"`
	StageIndex int    `json:"stage_index"`
	AttemptID  string `json:"attempt_id"`
	SessionID  string `json:"session_id"`
}

func reduceDispatchRequested(current state.State, event ingest.StateEvent) (state.State, []Effect, error) {
	var payload orderStagePayload
	if err := decodeEventPayload(event, &payload); err != nil {
		return current, nil, fmt.Errorf("reduce dispatch_requested: %w", err)
	}
	order, stage, ok := current.LookupStage(payload.OrderID, payload.StageIndex)
	if !ok || order.Status.IsTerminal() || stage.Status.IsTerminal() {
		return current, nil, nil
	}

	next := current.Clone()
	order = next.Orders[payload.OrderID]
	stage = order.Stages[payload.StageIndex]
	stage.Status = state.StageDispatching
	stage.Merge = nil
	order.Status = state.OrderActive
	order.UpdatedAt = event.Timestamp
	order.Stages[payload.StageIndex] = stage
	next.Orders[payload.OrderID] = order
	delete(next.PendingReviews, payload.OrderID)
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)

	effect, err := makeEffect(event, 0, EffectDispatch, map[string]any{
		"order_id":    payload.OrderID,
		"stage_index": payload.StageIndex,
		"attempt_id":  normalizedAttemptID(payload.AttemptID, event),
	})
	if err != nil {
		return current, nil, fmt.Errorf("reduce dispatch_requested: %w", err)
	}
	return next, []Effect{effect}, nil
}

func reduceDispatchCompleted(current state.State, event ingest.StateEvent) (state.State, []Effect, error) {
	var payload orderStagePayload
	if err := decodeEventPayload(event, &payload); err != nil {
		return current, nil, fmt.Errorf("reduce dispatch_completed: %w", err)
	}
	order, stage, ok := current.LookupStage(payload.OrderID, payload.StageIndex)
	if !ok || order.Status.IsTerminal() || stage.Status.IsTerminal() {
		return current, nil, nil
	}

	attemptID := normalizedAttemptID(payload.AttemptID, event)

	next := current.Clone()
	order = next.Orders[payload.OrderID]
	stage = order.Stages[payload.StageIndex]
	stage.Status = state.StageRunning
	stage.Merge = nil
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
	delete(next.PendingReviews, payload.OrderID)
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)
	return next, nil, nil
}

func reduceStageCompleted(current state.State, event ingest.StateEvent) (state.State, []Effect, error) {
	var payload orderStagePayload
	if err := decodeEventPayload(event, &payload); err != nil {
		return current, nil, fmt.Errorf("reduce stage_completed: %w", err)
	}
	order, stage, ok := current.LookupStage(payload.OrderID, payload.StageIndex)
	if !ok || order.Status.IsTerminal() || stage.Status.IsTerminal() {
		return current, nil, nil
	}

	next := current.Clone()
	order = next.Orders[payload.OrderID]
	stage = order.Stages[payload.StageIndex]
	order.UpdatedAt = event.Timestamp
	finalizeAttempt(&stage, payload, event, state.AttemptCompleted)

	if stageMergeable(stage, payload) {
		mergeRecovery, err := mergeRecoveryForStage(stage, payload)
		if err != nil {
			return current, nil, fmt.Errorf("reduce stage_completed merge recovery: %w", err)
		}
		stage.Status = state.StageMerging
		stage.Merge = mergeRecovery
		order.Stages[payload.StageIndex] = stage
		next.Orders[payload.OrderID] = order
		next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)

		effect, err := makeEffect(event, 0, EffectMerge, map[string]any{
			"order_id":      payload.OrderID,
			"stage_index":   payload.StageIndex,
			"worktree_name": mergeRecovery.WorktreeName,
			"merge_mode":    mergeRecovery.Mode,
			"merge_branch":  mergeRecovery.Branch,
		})
		if err != nil {
			return current, nil, fmt.Errorf("reduce stage_completed merge: %w", err)
		}
		return next, []Effect{effect}, nil
	}

	// Non-mergeable: complete immediately and check order advancement.
	stage.Status = state.StageCompleted
	stage.Merge = nil
	order.Stages[payload.StageIndex] = stage

	if allStagesTerminal(order) {
		order.Status = state.OrderCompleted
	}

	next.Orders[payload.OrderID] = order
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)

	if order.Status == state.OrderCompleted {
		e0, err := makeEffect(event, 0, EffectWriteProjection, map[string]any{"order_id": payload.OrderID})
		if err != nil {
			return current, nil, fmt.Errorf("reduce stage_completed projection: %w", err)
		}
		e1, err := makeEffect(event, 1, EffectAck, map[string]any{"order_id": payload.OrderID})
		if err != nil {
			return current, nil, fmt.Errorf("reduce stage_completed ack: %w", err)
		}
		return next, []Effect{e0, e1}, nil
	}
	return next, nil, nil
}

func reduceStageFailed(current state.State, event ingest.StateEvent) (state.State, []Effect, error) {
	var payload orderStagePayload
	if err := decodeEventPayload(event, &payload); err != nil {
		return current, nil, fmt.Errorf("reduce stage_failed: %w", err)
	}
	order, stage, ok := current.LookupStage(payload.OrderID, payload.StageIndex)
	if !ok || order.Status.IsTerminal() || stage.Status.IsTerminal() {
		return current, nil, nil
	}

	next := current.Clone()
	order = next.Orders[payload.OrderID]
	stage = order.Stages[payload.StageIndex]
	stage.Status = state.StageFailed
	stage.Merge = nil
	order.UpdatedAt = event.Timestamp
	finalizeAttempt(&stage, payload, event, state.AttemptFailed)
	order.Stages[payload.StageIndex] = stage
	order.Status = state.OrderFailed
	next.Orders[payload.OrderID] = order
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)

	effect, err := makeEffect(event, 0, EffectCleanup, map[string]any{
		"order_id":      payload.OrderID,
		"stage_index":   payload.StageIndex,
		"worktree_name": latestWorktreeName(stage),
	})
	if err != nil {
		return current, nil, fmt.Errorf("reduce stage_failed cleanup: %w", err)
	}
	return next, []Effect{effect}, nil
}

func reduceOrderCompleted(current state.State, event ingest.StateEvent) (state.State, []Effect, error) {
	var payload orderPayload
	if err := decodeEventPayload(event, &payload); err != nil {
		return current, nil, fmt.Errorf("reduce order_completed: %w", err)
	}
	order, ok := current.Orders[payload.OrderID]
	if !ok || order.Status.IsTerminal() {
		return current, nil, nil
	}

	next := current.Clone()
	order = next.Orders[payload.OrderID]
	order.Status = state.OrderCompleted
	order.UpdatedAt = event.Timestamp
	next.Orders[payload.OrderID] = order
	delete(next.PendingReviews, payload.OrderID)
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)

	e0, err := makeEffect(event, 0, EffectWriteProjection, map[string]any{"order_id": payload.OrderID})
	if err != nil {
		return current, nil, fmt.Errorf("reduce order_completed projection: %w", err)
	}
	e1, err := makeEffect(event, 1, EffectAck, map[string]any{"order_id": payload.OrderID})
	if err != nil {
		return current, nil, fmt.Errorf("reduce order_completed ack: %w", err)
	}
	return next, []Effect{e0, e1}, nil
}

func reduceOrderFailed(current state.State, event ingest.StateEvent) (state.State, []Effect, error) {
	var payload orderPayload
	if err := decodeEventPayload(event, &payload); err != nil {
		return current, nil, fmt.Errorf("reduce order_failed: %w", err)
	}
	order, ok := current.Orders[payload.OrderID]
	if !ok || order.Status.IsTerminal() {
		return current, nil, nil
	}

	next := current.Clone()
	order = next.Orders[payload.OrderID]
	order.Status = state.OrderFailed
	order.UpdatedAt = event.Timestamp
	next.Orders[payload.OrderID] = order
	delete(next.PendingReviews, payload.OrderID)
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)

	e0, err := makeEffect(event, 0, EffectWriteProjection, map[string]any{"order_id": payload.OrderID})
	if err != nil {
		return current, nil, fmt.Errorf("reduce order_failed projection: %w", err)
	}
	e1, err := makeEffect(event, 1, EffectAck, map[string]any{"order_id": payload.OrderID})
	if err != nil {
		return current, nil, fmt.Errorf("reduce order_failed ack: %w", err)
	}
	return next, []Effect{e0, e1}, nil
}

func reduceModeChanged(current state.State, event ingest.StateEvent) (state.State, []Effect, error) {
	var payload modePayload
	if err := decodeEventPayload(event, &payload); err != nil {
		return current, nil, fmt.Errorf("reduce mode_changed: %w", err)
	}
	mode, ok := parseRunMode(payload.Mode)
	if !ok {
		return current, nil, nil
	}

	next := current.Clone()
	oldMode := next.Mode
	next.Mode = mode
	next.ModeEpoch++

	record := state.ModeTransitionRecord{
		FromMode:    oldMode,
		ToMode:      mode,
		Epoch:       next.ModeEpoch,
		RequestedBy: strings.TrimSpace(payload.RequestedBy),
		Reason:      strings.TrimSpace(payload.Reason),
		AppliedAt:   event.Timestamp,
	}
	next.ModeTransitions = append(next.ModeTransitions, record)

	if len(next.ModeTransitions) > state.MaxModeTransitionHistory {
		next.ModeTransitions = next.ModeTransitions[len(next.ModeTransitions)-state.MaxModeTransitionHistory:]
	}

	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)
	return next, nil, nil
}

func reduceControlReceived(current state.State, event ingest.StateEvent) (state.State, []Effect, error) {
	var payload controlPayload
	if err := decodeEventPayload(event, &payload); err != nil {
		return current, nil, fmt.Errorf("reduce control_received: %w", err)
	}

	switch stringx.Normalize(payload.Command) {
	case "mode_change":
		mode, ok := parseRunMode(payload.Mode)
		if !ok {
			return current, nil, nil
		}
		next := current.Clone()
		oldMode := next.Mode
		next.Mode = mode
		next.ModeEpoch++

		record := state.ModeTransitionRecord{
			FromMode:    oldMode,
			ToMode:      mode,
			Epoch:       next.ModeEpoch,
			RequestedBy: "control",
			Reason:      "control_received",
			AppliedAt:   event.Timestamp,
		}
		next.ModeTransitions = append(next.ModeTransitions, record)

		if len(next.ModeTransitions) > state.MaxModeTransitionHistory {
			next.ModeTransitions = next.ModeTransitions[len(next.ModeTransitions)-state.MaxModeTransitionHistory:]
		}

		next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)
		return next, nil, nil
	default:
		// Unrecognized control command — no-op.
		return current, nil, nil
	}
}

func reduceSchedulePromoted(current state.State, event ingest.StateEvent) (state.State, []Effect, error) {
	var payload schedulePromotedPayload
	if err := decodeEventPayload(event, &payload); err != nil {
		return current, nil, fmt.Errorf("reduce schedule_promoted: %w", err)
	}
	orderID := strings.TrimSpace(payload.OrderID)
	if orderID == "" {
		return current, nil, nil
	}
	// Reject if order already exists.
	if _, exists := current.Orders[orderID]; exists {
		return current, nil, nil
	}

	next := current.Clone()
	if next.Orders == nil {
		next.Orders = make(map[string]state.OrderNode)
	}

	stages := payload.Stages
	if stages == nil {
		stages = []state.StageNode{}
	}
	// Ensure stage indexes are sequential starting at 0.
	for i := range stages {
		stages[i].StageIndex = i
		if stages[i].Status == "" {
			stages[i].Status = state.StagePending
		}
	}

	next.Orders[orderID] = state.OrderNode{
		OrderID:   orderID,
		Status:    state.OrderPending,
		Stages:    stages,
		CreatedAt: event.Timestamp,
		UpdatedAt: event.Timestamp,
		Metadata:  payload.Metadata,
	}
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)
	return next, nil, nil
}

func reduceSessionAdopted(current state.State, event ingest.StateEvent) (state.State, []Effect, error) {
	var payload sessionAdoptedPayload
	if err := decodeEventPayload(event, &payload); err != nil {
		return current, nil, fmt.Errorf("reduce session_adopted: %w", err)
	}
	order, stage, ok := current.LookupStage(payload.OrderID, payload.StageIndex)
	if !ok || order.Status.IsTerminal() || stage.Status.IsTerminal() {
		return current, nil, nil
	}

	attemptID := strings.TrimSpace(payload.AttemptID)
	sessionID := strings.TrimSpace(payload.SessionID)
	if attemptID == "" || sessionID == "" {
		return current, nil, nil
	}

	next := current.Clone()
	order = next.Orders[payload.OrderID]
	stage = order.Stages[payload.StageIndex]

	if idx, found := attemptIndexByID(stage.Attempts, attemptID); found {
		stage.Attempts[idx].SessionID = sessionID
		stage.Attempts[idx].Status = state.AttemptRunning
	} else {
		stage.Attempts = append(stage.Attempts, state.AttemptNode{
			AttemptID: attemptID,
			SessionID: sessionID,
			Status:    state.AttemptRunning,
			StartedAt: event.Timestamp,
		})
	}

	if stage.Status == state.StagePending || stage.Status == state.StageDispatching {
		stage.Status = state.StageRunning
	}

	order.Stages[payload.StageIndex] = stage
	order.UpdatedAt = event.Timestamp
	if order.Status == state.OrderPending {
		order.Status = state.OrderActive
	}
	next.Orders[payload.OrderID] = order
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)
	return next, nil, nil
}

func reduceMergeCompleted(current state.State, event ingest.StateEvent) (state.State, []Effect, error) {
	var payload orderStagePayload
	if err := decodeEventPayload(event, &payload); err != nil {
		return current, nil, fmt.Errorf("reduce merge_completed: %w", err)
	}
	order, stage, ok := current.LookupStage(payload.OrderID, payload.StageIndex)
	if !ok || order.Status.IsTerminal() {
		return current, nil, nil
	}
	if stage.Status != state.StageMerging {
		return current, nil, nil
	}

	next := current.Clone()
	order = next.Orders[payload.OrderID]
	stage = order.Stages[payload.StageIndex]
	stage.Status = state.StageCompleted
	stage.Merge = nil
	order.Stages[payload.StageIndex] = stage
	order.UpdatedAt = event.Timestamp

	if allStagesTerminal(order) {
		order.Status = state.OrderCompleted
	} else {
		order.Status = state.OrderActive
		nextStageIndex := payload.StageIndex + 1
		if nextStageIndex < len(order.Stages) && order.Stages[nextStageIndex].Status == "" {
			order.Stages[nextStageIndex].Status = state.StagePending
		}
	}

	next.Orders[payload.OrderID] = order
	delete(next.PendingReviews, payload.OrderID)
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)
	return next, nil, nil
}

func reduceMergeFailed(current state.State, event ingest.StateEvent) (state.State, []Effect, error) {
	var payload orderStagePayload
	if err := decodeEventPayload(event, &payload); err != nil {
		return current, nil, fmt.Errorf("reduce merge_failed: %w", err)
	}
	order, stage, ok := current.LookupStage(payload.OrderID, payload.StageIndex)
	if !ok || order.Status.IsTerminal() {
		return current, nil, nil
	}
	if stage.Status != state.StageMerging {
		return current, nil, nil
	}

	next := current.Clone()
	order = next.Orders[payload.OrderID]
	stage = order.Stages[payload.StageIndex]
	stage.Status = state.StageReview
	stage.Merge = nil
	order.UpdatedAt = event.Timestamp
	order.Stages[payload.StageIndex] = stage
	next.Orders[payload.OrderID] = order
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)

	effect, err := makeEffect(event, 0, EffectAck, map[string]any{
		"order_id":    payload.OrderID,
		"stage_index": payload.StageIndex,
	})
	if err != nil {
		return current, nil, fmt.Errorf("reduce merge_failed ack: %w", err)
	}
	return next, []Effect{effect}, nil
}

func reduceStageReviewParked(current state.State, event ingest.StateEvent) (state.State, []Effect, error) {
	var payload reviewPayload
	if err := decodeEventPayload(event, &payload); err != nil {
		return current, nil, fmt.Errorf("reduce stage_review_parked: %w", err)
	}
	order, stage, ok := current.LookupStage(payload.OrderID, payload.StageIndex)
	if !ok || order.Status.IsTerminal() {
		return current, nil, nil
	}

	next := current.Clone()
	if next.PendingReviews == nil {
		next.PendingReviews = make(map[string]state.PendingReviewNode)
	}
	order = next.Orders[payload.OrderID]
	stage = order.Stages[payload.StageIndex]
	stage.Status = state.StageReview
	stage.Merge = nil
	order.Status = state.OrderActive
	order.UpdatedAt = event.Timestamp
	recordReviewAttempt(&stage, payload, event)
	order.Stages[payload.StageIndex] = stage
	next.Orders[payload.OrderID] = order
	next.PendingReviews[payload.OrderID] = state.PendingReviewNode{
		OrderID:      payload.OrderID,
		StageIndex:   payload.StageIndex,
		TaskKey:      strings.TrimSpace(payload.TaskKey),
		Prompt:       strings.TrimSpace(payload.Prompt),
		Provider:     strings.TrimSpace(payload.Provider),
		Model:        strings.TrimSpace(payload.Model),
		Runtime:      strings.TrimSpace(payload.Runtime),
		Skill:        strings.TrimSpace(payload.Skill),
		Plan:         slices.Clone(payload.Plan),
		WorktreeName: strings.TrimSpace(payload.WorktreeName),
		WorktreePath: strings.TrimSpace(payload.WorktreePath),
		SessionID:    strings.TrimSpace(payload.SessionID),
		Reason:       strings.TrimSpace(payload.Reason),
	}
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)
	return next, nil, nil
}

func reduceStageReviewApproved(current state.State, event ingest.StateEvent) (state.State, []Effect, error) {
	var payload reviewPayload
	if err := decodeEventPayload(event, &payload); err != nil {
		return current, nil, fmt.Errorf("reduce stage_review_approved: %w", err)
	}
	if _, ok := current.PendingReviews[payload.OrderID]; !ok {
		return current, nil, nil
	}
	next := current.Clone()
	delete(next.PendingReviews, payload.OrderID)
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)
	return next, nil, nil
}

func reduceStageReviewChangesRequested(current state.State, event ingest.StateEvent) (state.State, []Effect, error) {
	var payload reviewPayload
	if err := decodeEventPayload(event, &payload); err != nil {
		return current, nil, fmt.Errorf("reduce stage_review_changes_requested: %w", err)
	}
	order, stage, ok := current.LookupStage(payload.OrderID, payload.StageIndex)
	if !ok || order.Status.IsTerminal() {
		return current, nil, nil
	}

	next := current.Clone()
	order = next.Orders[payload.OrderID]
	stage = order.Stages[payload.StageIndex]
	stage.Status = state.StageFailed
	stage.Merge = nil
	if len(stage.Attempts) > 0 {
		last := len(stage.Attempts) - 1
		stage.Attempts[last].Status = state.AttemptFailed
		stage.Attempts[last].Error = strings.TrimSpace(payload.Reason)
	}
	order.Stages[payload.StageIndex] = stage
	order.Status = state.OrderFailed
	order.UpdatedAt = event.Timestamp
	next.Orders[payload.OrderID] = order
	delete(next.PendingReviews, payload.OrderID)
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)
	return next, nil, nil
}

func reduceStageReviewRejected(current state.State, event ingest.StateEvent) (state.State, []Effect, error) {
	var payload reviewPayload
	if err := decodeEventPayload(event, &payload); err != nil {
		return current, nil, fmt.Errorf("reduce stage_review_rejected: %w", err)
	}
	order, _, ok := current.LookupStage(payload.OrderID, payload.StageIndex)
	if !ok || order.Status.IsTerminal() {
		return current, nil, nil
	}

	next := current.Clone()
	order = next.Orders[payload.OrderID]
	for i := range order.Stages {
		order.Stages[i].Merge = nil
		if order.Stages[i].Status.IsTerminal() {
			continue
		}
		order.Stages[i].Status = state.StageCancelled
	}
	order.Status = state.OrderCancelled
	order.UpdatedAt = event.Timestamp
	next.Orders[payload.OrderID] = order
	delete(next.PendingReviews, payload.OrderID)
	next.LastEventID = strconv.FormatUint(uint64(event.ID), 10)
	return next, nil, nil
}

func decodeEventPayload(event ingest.StateEvent, out any) error {
	if len(event.Payload) == 0 {
		return fmt.Errorf("event payload unavailable for type %q", event.Type)
	}
	if err := json.Unmarshal(event.Payload, out); err != nil {
		return fmt.Errorf("event payload unreadable for type %q: %w", event.Type, err)
	}
	return nil
}

func parseRunMode(raw string) (state.RunMode, bool) {
	normalized := stringx.Normalize(raw)
	switch state.RunMode(normalized) {
	case state.RunModeAuto, state.RunModeSupervised, state.RunModeManual:
		return state.RunMode(normalized), true
	default:
		return "", false
	}
}

func normalizedAttemptID(raw string, event ingest.StateEvent) string {
	id := strings.TrimSpace(raw)
	if id != "" {
		return id
	}
	return fmt.Sprintf("attempt-%d", event.ID)
}

func recordReviewAttempt(stage *state.StageNode, payload reviewPayload, event ingest.StateEvent) {
	attemptID := strings.TrimSpace(payload.AttemptID)
	if attemptID == "" {
		attemptID = fmt.Sprintf("review-%d", event.ID)
	}
	if idx, found := attemptIndexByID(stage.Attempts, attemptID); found {
		stage.Attempts[idx].SessionID = strings.TrimSpace(payload.SessionID)
		stage.Attempts[idx].Status = state.AttemptCompleted
		stage.Attempts[idx].CompletedAt = event.Timestamp
		if stage.Attempts[idx].WorktreeName == "" {
			stage.Attempts[idx].WorktreeName = strings.TrimSpace(payload.WorktreeName)
		}
		return
	}
	stage.Attempts = append(stage.Attempts, state.AttemptNode{
		AttemptID:    attemptID,
		SessionID:    strings.TrimSpace(payload.SessionID),
		Status:       state.AttemptCompleted,
		StartedAt:    event.Timestamp,
		CompletedAt:  event.Timestamp,
		WorktreeName: strings.TrimSpace(payload.WorktreeName),
	})
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

func mergeRecoveryForStage(stage state.StageNode, payload orderStagePayload) (*state.MergeRecoveryNode, error) {
	worktreeName := strings.TrimSpace(payload.WorktreeName)
	if worktreeName == "" {
		worktreeName = latestWorktreeName(stage)
	}
	if worktreeName == "" {
		return nil, fmt.Errorf("mergeable stage missing worktree name")
	}

	mergeMode := stringx.Normalize(payload.MergeMode)
	mergeBranch := strings.TrimSpace(payload.MergeBranch)
	switch mergeMode {
	case "":
		if mergeBranch != "" {
			mergeMode = "remote"
		} else {
			mergeMode = "local"
		}
	case "local":
		mergeBranch = ""
	case "remote":
		if mergeBranch == "" {
			return nil, fmt.Errorf("remote merge missing branch")
		}
	default:
		return nil, fmt.Errorf("invalid merge mode %q", payload.MergeMode)
	}

	return &state.MergeRecoveryNode{
		WorktreeName: worktreeName,
		Mode:         mergeMode,
		Branch:       mergeBranch,
	}, nil
}

func stageMergeable(stage state.StageNode, payload orderStagePayload) bool {
	if payload.Mergeable != nil {
		return *payload.Mergeable
	}
	return latestWorktreeName(stage) != ""
}

func allStagesTerminal(order state.OrderNode) bool {
	for _, stage := range order.Stages {
		if !stage.Status.IsTerminal() {
			return false
		}
	}
	return true
}

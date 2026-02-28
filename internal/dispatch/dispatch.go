package dispatch

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/rtcap"
	"github.com/poteto/noodle/internal/state"
)

type blockedReason string

const (
	blockedReasonBusy      blockedReason = "busy"
	blockedReasonCapacity  blockedReason = "capacity"
	blockedReasonFailed    blockedReason = "failed"
	blockedReasonNoPending blockedReason = "no pending stage"

	retryReasonOrderNotFound   string = "order not found"
	retryReasonStageNotFound   string = "stage not found"
	retryReasonOrderTerminal   string = "order is terminal"
	retryReasonStageTerminal   string = "stage is terminal"
	retryReasonRetryDisabled   string = "retry disabled"
	retryReasonAttemptsExhaust string = "max attempts reached"
	retryReasonPolicyRejected  string = "retry policy rejected completion"

	defaultRetryMaxAttempts = 2
)

type completionEventPayload struct {
	OrderID    string `json:"order_id"`
	StageIndex int    `json:"stage_index"`
	AttemptID  string `json:"attempt_id"`
	Error      string `json:"error"`
	ExitCode   *int   `json:"exit_code"`
}

// PlanDispatches scans canonical state and returns dispatch candidates plus
// blocked reasons.
func PlanDispatches(s state.State, maxConcurrent int, failedOrders map[string]bool) DispatchPlan {
	if maxConcurrent < 0 {
		maxConcurrent = 0
	}

	busyIndex := s.OrderBusyIndex()
	capacity := maxConcurrent - len(busyIndex)
	if capacity < 0 {
		capacity = 0
	}

	plan := DispatchPlan{
		Candidates:        make([]DispatchCandidate, 0),
		Blocked:           make([]BlockedCandidate, 0),
		CapacityRemaining: capacity,
	}

	orderIDs := sortedOrderIDs(s.Orders)
	for _, orderID := range orderIDs {
		order := s.Orders[orderID]
		if order.Status != state.OrderActive {
			continue
		}

		stageIndex, stage, ok := firstPendingStage(order)
		if !ok {
			plan.Blocked = append(plan.Blocked, BlockedCandidate{
				OrderID:    orderID,
				StageIndex: -1,
				Reason:     string(blockedReasonNoPending),
			})
			continue
		}

		if failedOrders != nil && failedOrders[orderID] {
			plan.Blocked = append(plan.Blocked, BlockedCandidate{
				OrderID:    orderID,
				StageIndex: stageIndex,
				Reason:     string(blockedReasonFailed),
			})
			continue
		}

		if _, busy := busyIndex[orderID]; busy {
			plan.Blocked = append(plan.Blocked, BlockedCandidate{
				OrderID:    orderID,
				StageIndex: stageIndex,
				Reason:     string(blockedReasonBusy),
			})
			continue
		}

		if len(plan.Candidates) >= capacity {
			plan.Blocked = append(plan.Blocked, BlockedCandidate{
				OrderID:    orderID,
				StageIndex: stageIndex,
				Reason:     string(blockedReasonCapacity),
			})
			continue
		}

		plan.Candidates = append(plan.Candidates, DispatchCandidate{
			OrderID:    orderID,
			StageIndex: stageIndex,
			Stage:      stage,
			Runtime:    resolvedRuntimeName(stage.Runtime),
		})
	}

	plan.CapacityRemaining = capacity - len(plan.Candidates)
	if plan.CapacityRemaining < 0 {
		plan.CapacityRemaining = 0
	}
	return plan
}

// RouteCompletion applies one attempt completion and emits canonical routing
// events for downstream ingestion.
func RouteCompletion(s state.State, rec CompletionRecord) (state.State, []ingest.StateEvent) {
	order, stage, ok := lookupOrderStage(s, rec.OrderID, rec.StageIndex)
	if !ok || isTerminalOrder(order.Status) {
		return s, nil
	}
	if isTerminalStage(stage.Status) {
		return s, nil
	}

	next := cloneState(s)
	order = next.Orders[rec.OrderID]
	stage = order.Stages[rec.StageIndex]
	updateAttemptFromCompletion(&stage, rec)

	if !rec.CompletedAt.IsZero() {
		order.UpdatedAt = rec.CompletedAt
	}

	events := make([]ingest.StateEvent, 0, 1)
	switch rec.Status {
	case state.AttemptCompleted:
		stage.Status = state.StageCompleted
		order.Status = state.OrderActive
		order.Stages[rec.StageIndex] = stage
		next.Orders[rec.OrderID] = order

		events = append(events, buildInternalEvent(ingest.EventStageCompleted, rec.CompletedAt, completionEventPayload{
			OrderID:    rec.OrderID,
			StageIndex: rec.StageIndex,
			AttemptID:  rec.AttemptID,
			Error:      strings.TrimSpace(rec.Error),
			ExitCode:   clonedExitCode(rec.ExitCode),
		}))

	case state.AttemptFailed, state.AttemptCancelled:
		order.Stages[rec.StageIndex] = stage
		next.Orders[rec.OrderID] = order

		canRetry, _ := RetryCandidate(next, rec.OrderID, rec.StageIndex, defaultRetryPolicy())
		if canRetry {
			order = next.Orders[rec.OrderID]
			stage = order.Stages[rec.StageIndex]
			stage.Status = state.StagePending
			order.Status = state.OrderActive
			order.Stages[rec.StageIndex] = stage
			next.Orders[rec.OrderID] = order

			events = append(events, buildInternalEvent(ingest.EventDispatchRequested, rec.CompletedAt, completionEventPayload{
				OrderID:    rec.OrderID,
				StageIndex: rec.StageIndex,
				AttemptID:  rec.AttemptID,
				Error:      strings.TrimSpace(rec.Error),
				ExitCode:   clonedExitCode(rec.ExitCode),
			}))
			return next, events
		}

		next = RouteFailure(next, rec.OrderID, rec.StageIndex, rec.Error)
		events = append(events, buildInternalEvent(ingest.EventStageFailed, rec.CompletedAt, completionEventPayload{
			OrderID:    rec.OrderID,
			StageIndex: rec.StageIndex,
			AttemptID:  rec.AttemptID,
			Error:      strings.TrimSpace(rec.Error),
			ExitCode:   clonedExitCode(rec.ExitCode),
		}))
	default:
		order.Stages[rec.StageIndex] = stage
		next.Orders[rec.OrderID] = order
	}

	return next, events
}

// AdvanceOrder marks post-merge progress and returns removed=true when the
// order reached completion.
func AdvanceOrder(s state.State, orderID string) (state.State, bool) {
	order, ok := s.Orders[orderID]
	if !ok || isTerminalOrder(order.Status) {
		return s, false
	}

	next := cloneState(s)
	order = next.Orders[orderID]

	for i := range order.Stages {
		if order.Stages[i].Status == state.StageMerging {
			order.Stages[i].Status = state.StageCompleted
			break
		}
	}

	for i := range order.Stages {
		if isPendingStage(order.Stages[i].Status) {
			order.Stages[i].Status = state.StagePending
			break
		}
	}

	if len(order.Stages) == 0 || allStagesCompleted(order.Stages) {
		order.Status = state.OrderCompleted
		next.Orders[orderID] = order
		return next, true
	}

	if anyStageFailed(order.Stages) {
		order.Status = state.OrderFailed
		next.Orders[orderID] = order
		return next, false
	}

	order.Status = state.OrderActive
	next.Orders[orderID] = order
	return next, false
}

// RouteFailure marks stage failure and marks order failure when retry is no
// longer possible.
func RouteFailure(s state.State, orderID string, stageIndex int, reason string) state.State {
	order, stage, ok := lookupOrderStage(s, orderID, stageIndex)
	if !ok || isTerminalOrder(order.Status) {
		return s
	}

	next := cloneState(s)
	order = next.Orders[orderID]
	stage = order.Stages[stageIndex]
	stage.Status = state.StageFailed

	if len(stage.Attempts) > 0 {
		last := len(stage.Attempts) - 1
		stage.Attempts[last].Status = state.AttemptFailed
		stage.Attempts[last].Error = strings.TrimSpace(reason)
	}

	order.Stages[stageIndex] = stage
	next.Orders[orderID] = order

	canRetry, _ := RetryCandidate(next, orderID, stageIndex, defaultRetryPolicy())
	if canRetry {
		order.Status = state.OrderActive
	} else {
		order.Status = state.OrderFailed
	}
	next.Orders[orderID] = order
	return next
}

// RetryCandidate reports whether a stage can retry under the provided policy.
func RetryCandidate(s state.State, orderID string, stageIndex int, policy RetryPolicy) (bool, string) {
	order, stage, ok := lookupOrderStage(s, orderID, stageIndex)
	if !ok {
		if _, exists := s.Orders[orderID]; !exists {
			return false, retryReasonOrderNotFound
		}
		return false, retryReasonStageNotFound
	}

	if isTerminalOrder(order.Status) {
		return false, retryReasonOrderTerminal
	}

	if isTerminalStage(stage.Status) && stage.Status != state.StageFailed {
		return false, retryReasonStageTerminal
	}

	if policy.MaxAttempts <= 0 {
		return false, retryReasonRetryDisabled
	}

	if len(stage.Attempts) >= policy.MaxAttempts {
		return false, retryReasonAttemptsExhaust
	}

	if policy.ShouldRetry != nil {
		rec := latestCompletionRecord(orderID, stageIndex, stage)
		if !policy.ShouldRetry(rec) {
			return false, retryReasonPolicyRejected
		}
	}
	return true, ""
}

func sortedOrderIDs(orders map[string]state.OrderNode) []string {
	ids := make([]string, 0, len(orders))
	for orderID := range orders {
		ids = append(ids, orderID)
	}
	slices.Sort(ids)
	return ids
}

func firstPendingStage(order state.OrderNode) (int, state.StageNode, bool) {
	for i := range order.Stages {
		if isPendingStage(order.Stages[i].Status) {
			return i, order.Stages[i], true
		}
	}
	return -1, state.StageNode{}, false
}

func isPendingStage(status state.StageLifecycleStatus) bool {
	return status == "" || status == state.StagePending
}

func resolvedRuntimeName(raw string) string {
	name := strings.ToLower(strings.TrimSpace(raw))
	if name == "" {
		return rtcap.ProcessCaps.Name
	}
	return name
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

func allStagesCompleted(stages []state.StageNode) bool {
	for _, stage := range stages {
		switch stage.Status {
		case state.StageCompleted, state.StageSkipped, state.StageCancelled:
			continue
		default:
			return false
		}
	}
	return true
}

func anyStageFailed(stages []state.StageNode) bool {
	for _, stage := range stages {
		if stage.Status == state.StageFailed {
			return true
		}
	}
	return false
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

func updateAttemptFromCompletion(stage *state.StageNode, rec CompletionRecord) {
	attemptID := strings.TrimSpace(rec.AttemptID)
	idx := -1
	if attemptID != "" {
		for i := range stage.Attempts {
			if stage.Attempts[i].AttemptID == attemptID {
				idx = i
				break
			}
		}
	}

	if idx < 0 {
		if attemptID == "" && len(stage.Attempts) > 0 {
			idx = len(stage.Attempts) - 1
		} else {
			stage.Attempts = append(stage.Attempts, state.AttemptNode{AttemptID: attemptID})
			idx = len(stage.Attempts) - 1
		}
	}

	attempt := stage.Attempts[idx]
	if attemptID != "" {
		attempt.AttemptID = attemptID
	}
	attempt.Status = rec.Status
	if !rec.CompletedAt.IsZero() {
		attempt.CompletedAt = rec.CompletedAt
	}
	if rec.ExitCode != nil {
		attempt.ExitCode = clonedExitCode(rec.ExitCode)
	}
	if trimmed := strings.TrimSpace(rec.Error); trimmed != "" {
		attempt.Error = trimmed
	}
	stage.Attempts[idx] = attempt
}

func latestCompletionRecord(orderID string, stageIndex int, stage state.StageNode) CompletionRecord {
	if len(stage.Attempts) == 0 {
		return CompletionRecord{
			OrderID:    orderID,
			StageIndex: stageIndex,
		}
	}
	last := stage.Attempts[len(stage.Attempts)-1]
	return CompletionRecord{
		OrderID:     orderID,
		StageIndex:  stageIndex,
		AttemptID:   last.AttemptID,
		Status:      last.Status,
		ExitCode:    clonedExitCode(last.ExitCode),
		Error:       last.Error,
		CompletedAt: last.CompletedAt,
	}
}

func defaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: defaultRetryMaxAttempts,
		ShouldRetry: func(rec CompletionRecord) bool {
			return rec.Status == state.AttemptFailed
		},
	}
}

func buildInternalEvent(eventType ingest.EventType, timestamp time.Time, payload completionEventPayload) ingest.StateEvent {
	data, err := json.Marshal(payload)
	if err != nil {
		// dispatch-owned payloads are deterministic structs.
		panic(fmt.Sprintf("encode dispatch event payload: %v", err))
	}
	return ingest.StateEvent{
		Source:         string(ingest.SourceInternal),
		Type:           string(eventType),
		Timestamp:      timestamp,
		Payload:        data,
		IdempotencyKey: fmt.Sprintf("%s:%s:%d:%s", eventType, payload.OrderID, payload.StageIndex, payload.AttemptID),
	}
}

func clonedExitCode(code *int) *int {
	if code == nil {
		return nil
	}
	v := *code
	return &v
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
			metadata := make(map[string]string, len(order.Metadata))
			for key, value := range order.Metadata {
				metadata[key] = value
			}
			orderCopy.Metadata = metadata
		}

		if order.Stages != nil {
			stages := make([]state.StageNode, len(order.Stages))
			for i := range order.Stages {
				stageCopy := order.Stages[i]
				if order.Stages[i].Attempts != nil {
					attempts := make([]state.AttemptNode, len(order.Stages[i].Attempts))
					for j := range order.Stages[i].Attempts {
						attemptCopy := order.Stages[i].Attempts[j]
						attemptCopy.ExitCode = clonedExitCode(attemptCopy.ExitCode)
						attempts[j] = attemptCopy
					}
					stageCopy.Attempts = attempts
				}
				stages[i] = stageCopy
			}
			orderCopy.Stages = stages
		}

		out.Orders[orderID] = orderCopy
	}
	return out
}

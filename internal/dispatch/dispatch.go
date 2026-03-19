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
	"github.com/poteto/noodle/internal/stringx"
)

type blockedReason string

const (
	blockedReasonBusy      blockedReason = "busy"
	blockedReasonCapacity  blockedReason = "capacity"
	blockedReasonFailed    blockedReason = "failed"
	blockedReasonTicketed  blockedReason = "ticketed"
	blockedReasonReview    blockedReason = "pending_review"
	blockedReasonNoPending blockedReason = "no pending stage"
)

type completionEventPayload struct {
	OrderID    string `json:"order_id"`
	StageIndex int    `json:"stage_index"`
	AttemptID  string `json:"attempt_id"`
	Error      string `json:"error"`
	ExitCode   *int   `json:"exit_code"`
}

// PlanDispatches scans canonical state and returns dispatch candidates plus
// blocked reasons. blockedOrders are external blockers such as ticket ownership
// or pending review that should prevent dispatch before capacity is applied.
func PlanDispatches(s state.State, maxConcurrent int, blockedOrders map[string]string) DispatchPlan {
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
		if order.Status.IsTerminal() && order.Status != state.OrderFailed {
			continue
		}

		stageIndex, stage, ok := firstPendingStage(order)
		if order.Status == state.OrderFailed {
			if !ok {
				stageIndex = firstBlockedStageIndex(order)
			}
			plan.Blocked = append(plan.Blocked, BlockedCandidate{
				OrderID:    orderID,
				StageIndex: stageIndex,
				Reason:     string(blockedReasonFailed),
			})
			continue
		}

		if order.Status != state.OrderPending && order.Status != state.OrderActive {
			continue
		}
		if !ok {
			plan.Blocked = append(plan.Blocked, BlockedCandidate{
				OrderID:    orderID,
				StageIndex: -1,
				Reason:     string(blockedReasonNoPending),
			})
			continue
		}

		if blockedOrders != nil {
			if reason := strings.TrimSpace(blockedOrders[orderID]); reason != "" {
				plan.Blocked = append(plan.Blocked, BlockedCandidate{
					OrderID:    orderID,
					StageIndex: stageIndex,
					Reason:     reason,
				})
				continue
			}
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
func RouteCompletion(s state.State, rec CompletionRecord) (state.State, []ingest.StateEvent, error) {
	order, stage, ok := s.LookupStage(rec.OrderID, rec.StageIndex)
	if !ok || order.Status.IsTerminal() {
		return s, nil, nil
	}
	if stage.Status.IsTerminal() {
		return s, nil, nil
	}

	next := s.Clone()
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

		evt, err := buildInternalEvent(ingest.EventStageCompleted, rec.CompletedAt, completionEventPayload{
			OrderID:    rec.OrderID,
			StageIndex: rec.StageIndex,
			AttemptID:  rec.AttemptID,
			Error:      strings.TrimSpace(rec.Error),
			ExitCode:   state.ClonedExitCode(rec.ExitCode),
		})
		if err != nil {
			return s, nil, fmt.Errorf("route completion stage_completed: %w", err)
		}
		events = append(events, evt)

	case state.AttemptFailed, state.AttemptCancelled:
		order.Stages[rec.StageIndex] = stage
		next.Orders[rec.OrderID] = order

		next = RouteFailure(next, rec.OrderID, rec.StageIndex, rec.Error)
		evt, err := buildInternalEvent(ingest.EventStageFailed, rec.CompletedAt, completionEventPayload{
			OrderID:    rec.OrderID,
			StageIndex: rec.StageIndex,
			AttemptID:  rec.AttemptID,
			Error:      strings.TrimSpace(rec.Error),
			ExitCode:   state.ClonedExitCode(rec.ExitCode),
		})
		if err != nil {
			return s, nil, fmt.Errorf("route completion stage_failed: %w", err)
		}
		events = append(events, evt)
	default:
		order.Stages[rec.StageIndex] = stage
		next.Orders[rec.OrderID] = order
	}

	return next, events, nil
}

// AdvanceOrder marks post-merge progress and returns removed=true when the
// order reached completion.
func AdvanceOrder(s state.State, orderID string) (state.State, bool) {
	order, ok := s.Orders[orderID]
	if !ok || order.Status.IsTerminal() {
		return s, false
	}

	next := s.Clone()
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
	order, stage, ok := s.LookupStage(orderID, stageIndex)
	if !ok || order.Status.IsTerminal() {
		return s
	}

	next := s.Clone()
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

	order.Status = state.OrderFailed
	next.Orders[orderID] = order
	return next
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

func firstBlockedStageIndex(order state.OrderNode) int {
	for i := range order.Stages {
		if !order.Stages[i].Status.IsTerminal() {
			return order.Stages[i].StageIndex
		}
	}
	if len(order.Stages) == 0 {
		return -1
	}
	return order.Stages[0].StageIndex
}

func isPendingStage(status state.StageLifecycleStatus) bool {
	return status == "" || status == state.StagePending
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

func resolvedRuntimeName(raw string) string {
	name := stringx.Normalize(raw)
	if name == "" {
		return rtcap.ProcessCaps.Name
	}
	return name
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
		attempt.ExitCode = state.ClonedExitCode(rec.ExitCode)
	}
	if trimmed := strings.TrimSpace(rec.Error); trimmed != "" {
		attempt.Error = trimmed
	}
	stage.Attempts[idx] = attempt
}

func buildInternalEvent(eventType ingest.EventType, timestamp time.Time, payload completionEventPayload) (ingest.StateEvent, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return ingest.StateEvent{}, fmt.Errorf("dispatch event payload encoding failed: %v", err)
	}
	return ingest.StateEvent{
		Source:         string(ingest.SourceInternal),
		Type:           string(eventType),
		Timestamp:      timestamp,
		Payload:        data,
		IdempotencyKey: fmt.Sprintf("%s:%s:%d:%s", eventType, payload.OrderID, payload.StageIndex, payload.AttemptID),
	}, nil
}

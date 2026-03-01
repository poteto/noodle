package loop

import (
	"strings"

	"github.com/poteto/noodle/internal/failure"
)

// SchedulerMistakeReason identifies scheduler-agent owned mistakes.
type SchedulerMistakeReason string

const (
	SchedulerMistakeReasonOrdersNextRejected SchedulerMistakeReason = "orders_next_rejected"
)

// CookMistakeReason identifies cook-agent owned mistakes.
type CookMistakeReason string

const (
	CookMistakeReasonQualityRejected CookMistakeReason = "quality_rejected"
	CookMistakeReasonReviewRejected  CookMistakeReason = "review_rejected"
	CookMistakeReasonRequestChanges  CookMistakeReason = "request_changes"
)

// AgentMistakeEnvelope captures agent ownership classification for failures that
// are not backend invariant/runtime faults.
type AgentMistakeEnvelope struct {
	FailureClass    failure.FailureClass          `json:"failure_class"`
	Owner           failure.FailureOwner          `json:"owner"`
	Scope           failure.FailureScope          `json:"scope"`
	Recoverability  failure.FailureRecoverability `json:"recoverability"`
	SchedulerReason SchedulerMistakeReason        `json:"scheduler_reason,omitempty"`
	CookReason      CookMistakeReason             `json:"cook_reason,omitempty"`
	OrderID         string                        `json:"order_id,omitempty"`
	StageIndex      *int                          `json:"stage_index,omitempty"`
}

func newSchedulerMistakeEnvelope(reason SchedulerMistakeReason) AgentMistakeEnvelope {
	failureClass := failure.FailureClassAgentMistake
	return AgentMistakeEnvelope{
		FailureClass:    failureClass,
		Owner:           failure.FailureOwnerSchedulerAgent,
		Scope:           failure.FailureScopeSystem,
		Recoverability:  failure.RecoverabilityForClass(failureClass),
		SchedulerReason: reason,
	}
}

func newCookMistakeEnvelope(reason CookMistakeReason, orderID string, stageIndex int) AgentMistakeEnvelope {
	failureClass := failure.FailureClassAgentMistake
	envelope := AgentMistakeEnvelope{
		FailureClass:   failureClass,
		Owner:          failure.FailureOwnerCookAgent,
		Scope:          failure.FailureScopeOrder,
		Recoverability: failure.RecoverabilityForClass(failureClass),
		CookReason:     reason,
		OrderID:        strings.TrimSpace(orderID),
	}
	if stageIndex >= 0 {
		stage := stageIndex
		envelope.StageIndex = &stage
	}
	return envelope
}

func cookRejectReasonForTask(taskKey string) CookMistakeReason {
	if strings.EqualFold(strings.TrimSpace(taskKey), "quality") {
		return CookMistakeReasonQualityRejected
	}
	return CookMistakeReasonReviewRejected
}

func agentMistakeReason(envelope *AgentMistakeEnvelope) string {
	if envelope == nil {
		return ""
	}
	if envelope.SchedulerReason != "" {
		return string(envelope.SchedulerReason)
	}
	if envelope.CookReason != "" {
		return string(envelope.CookReason)
	}
	return ""
}

func cloneAgentMistakeEnvelope(envelope *AgentMistakeEnvelope) *AgentMistakeEnvelope {
	if envelope == nil {
		return nil
	}
	cloned := *envelope
	if envelope.StageIndex != nil {
		stage := *envelope.StageIndex
		cloned.StageIndex = &stage
	}
	return &cloned
}

package loop

import "github.com/poteto/noodle/event"

// Re-export loop event types for in-package use.
const (
	LoopEventStageCompleted     = event.LoopEventStageCompleted
	LoopEventStageFailed        = event.LoopEventStageFailed
	LoopEventOrderCompleted     = event.LoopEventOrderCompleted
	LoopEventOrderFailed        = event.LoopEventOrderFailed
	LoopEventOrderDropped       = event.LoopEventOrderDropped
	LoopEventOrderRequeued      = event.LoopEventOrderRequeued
	LoopEventWorktreeMerged     = event.LoopEventWorktreeMerged
	LoopEventMergeConflict      = event.LoopEventMergeConflict
	LoopEventScheduleCompleted  = event.LoopEventScheduleCompleted
	LoopEventRegistryRebuilt    = event.LoopEventRegistryRebuilt
	LoopEventSyncDegraded       = event.LoopEventSyncDegraded
	LoopEventBootstrapCompleted = event.LoopEventBootstrapCompleted
	LoopEventBootstrapExhausted = event.LoopEventBootstrapExhausted
	LoopEventPromotionFailed    = event.LoopEventPromotionFailed
)

// Payload structs for loop events.

type RegistryRebuiltPayload struct {
	Added   []string `json:"added,omitempty"`
	Removed []string `json:"removed,omitempty"`
}

type BootstrapExhaustedPayload struct {
	Reason  string                `json:"reason"`
	Failure *EventFailureMetadata `json:"failure,omitempty"`
}

type OrderDroppedPayload struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

type SyncDegradedPayload struct {
	Reason  string                `json:"reason"`
	Failure *EventFailureMetadata `json:"failure,omitempty"`
}

type StageCompletedPayload struct {
	OrderID    string  `json:"order_id"`
	StageIndex int     `json:"stage_index"`
	TaskKey    string  `json:"task_key"`
	SessionID  *string `json:"session_id,omitempty"`
	Message    *string `json:"message,omitempty"`
}

type StageFailedPayload struct {
	OrderID      string                `json:"order_id"`
	StageIndex   int                   `json:"stage_index"`
	Reason       string                `json:"reason"`
	SessionID    *string               `json:"session_id,omitempty"`
	AgentMistake *AgentMistakeEnvelope `json:"agent_mistake,omitempty"`
	Failure      *EventFailureMetadata `json:"failure,omitempty"`
}

type OrderCompletedPayload struct {
	OrderID string `json:"order_id"`
}

type OrderFailedPayload struct {
	OrderID      string                `json:"order_id"`
	Reason       string                `json:"reason"`
	AgentMistake *AgentMistakeEnvelope `json:"agent_mistake,omitempty"`
	Failure      *EventFailureMetadata `json:"failure,omitempty"`
}

type ScheduleCompletedPayload struct {
	SessionID string `json:"session_id"`
}

type WorktreeMergedPayload struct {
	OrderID      string `json:"order_id"`
	StageIndex   int    `json:"stage_index"`
	WorktreeName string `json:"worktree_name"`
}

type MergeConflictPayload struct {
	OrderID      string `json:"order_id"`
	StageIndex   int    `json:"stage_index"`
	WorktreeName string `json:"worktree_name"`
}

type OrderRequeuedPayload struct {
	OrderID string `json:"order_id"`
}

type PromotionFailedPayload struct {
	Reason       string                `json:"reason"`
	AgentMistake *AgentMistakeEnvelope `json:"agent_mistake,omitempty"`
	Failure      *EventFailureMetadata `json:"failure,omitempty"`
}

// sessionIDPtr returns a pointer to the session ID, or nil if the cook has no session.
func sessionIDPtr(cook *cookHandle) *string {
	if cook.session == nil {
		return nil
	}
	id := cook.session.ID()
	return &id
}

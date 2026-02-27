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
	LoopEventQualityWritten     = event.LoopEventQualityWritten
	LoopEventScheduleCompleted  = event.LoopEventScheduleCompleted
	LoopEventRegistryRebuilt    = event.LoopEventRegistryRebuilt
	LoopEventSyncDegraded       = event.LoopEventSyncDegraded
	LoopEventBootstrapCompleted = event.LoopEventBootstrapCompleted
	LoopEventBootstrapExhausted = event.LoopEventBootstrapExhausted
)

// Payload structs for loop events.

type RegistryRebuiltPayload struct {
	Added   []string `json:"added,omitempty"`
	Removed []string `json:"removed,omitempty"`
}

type BootstrapExhaustedPayload struct {
	Reason string `json:"reason"`
}

type OrderDroppedPayload struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

type SyncDegradedPayload struct {
	Reason string `json:"reason"`
}

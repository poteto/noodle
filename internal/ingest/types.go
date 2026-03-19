package ingest

import (
	"encoding/json"
	"time"
)

// EventID is the ingestion sequence number assigned by the ingestion arbiter.
type EventID uint64

// EventSource identifies the ingress source for a canonical event.
type EventSource string

const (
	SourceControl   EventSource = "control"
	SourceScheduler EventSource = "scheduler"
	SourceRuntime   EventSource = "runtime"
	SourceInternal  EventSource = "internal"
)

// EventType identifies the canonical event type.
type EventType string

const (
	EventControlReceived   EventType = "control_received"
	EventSchedulePromoted  EventType = "schedule_promoted"
	EventDispatchRequested EventType = "dispatch_requested"
	EventDispatchCompleted EventType = "dispatch_completed"

	EventStageCompleted              EventType = "stage_completed"
	EventStageFailed                 EventType = "stage_failed"
	EventStageReviewParked           EventType = "stage_review_parked"
	EventStageReviewApproved         EventType = "stage_review_approved"
	EventStageReviewChangesRequested EventType = "stage_review_changes_requested"
	EventStageReviewRejected         EventType = "stage_review_rejected"
	EventOrderCompleted              EventType = "order_completed"
	EventOrderFailed                 EventType = "order_failed"

	EventModeChanged    EventType = "mode_changed"
	EventSessionAdopted EventType = "session_adopted"
	EventMergeCompleted EventType = "merge_completed"
	EventMergeFailed    EventType = "merge_failed"
)

var allEventTypes = []EventType{
	EventControlReceived,
	EventSchedulePromoted,
	EventDispatchRequested,
	EventDispatchCompleted,
	EventStageCompleted,
	EventStageFailed,
	EventStageReviewParked,
	EventStageReviewApproved,
	EventStageReviewChangesRequested,
	EventStageReviewRejected,
	EventOrderCompleted,
	EventOrderFailed,
	EventModeChanged,
	EventSessionAdopted,
	EventMergeCompleted,
	EventMergeFailed,
}

var knownEventTypeSet = func() map[EventType]struct{} {
	index := make(map[EventType]struct{}, len(allEventTypes))
	for _, eventType := range allEventTypes {
		index[eventType] = struct{}{}
	}
	return index
}()

// AllEventTypes returns a copy of the canonical event type registry.
func AllEventTypes() []EventType {
	out := make([]EventType, len(allEventTypes))
	copy(out, allEventTypes)
	return out
}

// IsKnownEventType reports whether eventType is part of the canonical registry.
func IsKnownEventType(eventType EventType) bool {
	_, ok := knownEventTypeSet[eventType]
	return ok
}

// StateEvent is the canonical state-event envelope.
type StateEvent struct {
	ID             EventID         `json:"id"`
	Source         string          `json:"source"`
	Type           string          `json:"type"`
	Timestamp      time.Time       `json:"timestamp"`
	Payload        json.RawMessage `json:"payload"`
	IdempotencyKey string          `json:"idempotency_key"`
	DedupReason    string          `json:"dedup_reason"`
	Applied        bool            `json:"applied"`
}

// InputEnvelope is raw external input before normalization.
type InputEnvelope struct {
	Source     string          `json:"source"`
	RawPayload json.RawMessage `json:"raw_payload"`
	ReceivedAt time.Time       `json:"received_at"`
}

// IngestStats tracks ingestion counters.
type IngestStats struct {
	TotalEvents   uint64 `json:"total_events"`
	DedupedEvents uint64 `json:"deduped_events"`
	AppliedEvents uint64 `json:"applied_events"`
}

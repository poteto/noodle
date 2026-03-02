package event

import (
	"encoding/json"
	"time"
)

// EventType is the normalized event type used in session event logs.
type EventType string

const (
	// Core runtime events.
	EventSpawned     EventType = "spawned"
	EventAction      EventType = "action"
	EventCost        EventType = "cost"
	EventStateChange EventType = "state_change"
	EventExited      EventType = "exited"
	EventDelta       EventType = "delta"

	// Cook lifecycle events.
	EventCookStarted   EventType = "cook_started"
	EventCookCompleted EventType = "cook_completed"

	// Stage message events (emitted by agents via CLI).
	EventStageMessage EventType = "stage_message"

	// Ticket protocol events.
	EventTicketClaim    EventType = "ticket_claim"
	EventTicketProgress EventType = "ticket_progress"
	EventTicketDone     EventType = "ticket_done"
	EventTicketBlocked  EventType = "ticket_blocked"
	EventTicketRelease  EventType = "ticket_release"
)

// Event is one append-only NDJSON record in a session event log.
type Event struct {
	Type      EventType       `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	SessionID string          `json:"session_id"`
}

// StageMessagePayload is the payload for stage_message events.
type StageMessagePayload struct {
	Message  string `json:"message"`
	Blocking *bool  `json:"blocking,omitempty"`
}

// IsBlocking returns whether the message blocks auto-advance.
// Default (nil) is true — blocking.
func (p StageMessagePayload) IsBlocking() bool {
	if p.Blocking == nil {
		return true
	}
	return *p.Blocking
}

// TicketTargetType identifies what a ticket claims.
type TicketTargetType string

const (
	TicketTargetBacklogItem TicketTargetType = "backlog_item"
	TicketTargetFile        TicketTargetType = "file"
	TicketTargetPlanPhase   TicketTargetType = "plan_phase"
)

// TicketStatus is the materialized ticket state.
type TicketStatus string

const (
	TicketStatusActive  TicketStatus = "active"
	TicketStatusBlocked TicketStatus = "blocked"
)

// Ticket is the current materialized claim for one target.
type Ticket struct {
	Target       string           `json:"target"`
	TargetType   TicketTargetType `json:"target_type"`
	CookID       string           `json:"cook_id"`
	Files        []string         `json:"files,omitempty"`
	ClaimedAt    time.Time        `json:"claimed_at"`
	LastProgress time.Time        `json:"last_progress"`
	Status       TicketStatus     `json:"status"`
	BlockedBy    string           `json:"blocked_by,omitempty"`
	Reason       string           `json:"reason,omitempty"`
}

// TicketEventPayload is the payload contract used by ticket protocol events.
type TicketEventPayload struct {
	Target     string           `json:"target"`
	TargetType TicketTargetType `json:"target_type"`
	Files      []string         `json:"files,omitempty"`
	Summary    string           `json:"summary,omitempty"`
	Outcome    string           `json:"outcome,omitempty"`
	BlockedBy  string           `json:"blocked_by,omitempty"`
	Reason     string           `json:"reason,omitempty"`
}

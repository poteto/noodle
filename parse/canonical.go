package parse

import "time"

// EventType is the normalized event kind emitted across providers.
type EventType string

const (
	EventInit     EventType = "init"
	EventAction   EventType = "action"
	EventResult   EventType = "result"
	EventError    EventType = "error"
	EventComplete EventType = "complete"
)

// CanonicalEvent is the provider-agnostic event schema.
type CanonicalEvent struct {
	Provider  string    `json:"provider,omitempty"`
	Type      EventType `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	CostUSD   float64   `json:"cost_usd,omitempty"`
	TokensIn  int       `json:"tokens_in,omitempty"`
	TokensOut int       `json:"tokens_out,omitempty"`
}

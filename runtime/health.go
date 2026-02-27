package runtime

import "time"

// HealthEventType classifies the health state of a session.
type HealthEventType string

const (
	HealthHealthy HealthEventType = "healthy"
	HealthIdle    HealthEventType = "idle"
	HealthStuck   HealthEventType = "stuck"
	HealthDead    HealthEventType = "dead"
)

// HealthEvent reports a session's health state change. Each event carries a
// monotonic sequence number per session to prevent stale events from
// regressing state.
type HealthEvent struct {
	SessionID string          `json:"session_id"`
	OrderID   string          `json:"order_id"`
	Type      HealthEventType `json:"type"`
	Detail    string          `json:"detail,omitempty"`
	At        time.Time       `json:"at"`
	Seq       uint64          `json:"seq"`
}

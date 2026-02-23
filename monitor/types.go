package monitor

import (
	"context"
	"time"
)

const (
	defaultStuckThreshold = 120 * time.Second
	defaultPollInterval   = 5 * time.Second
	defaultDebounce       = 200 * time.Millisecond
	contextTokenBudget    = 200000.0
)

type SessionStatus string

const (
	SessionStatusRunning SessionStatus = "running"
	SessionStatusStuck   SessionStatus = "stuck"
	SessionStatusExited  SessionStatus = "exited"
	SessionStatusFailed  SessionStatus = "failed"
)

type HealthStatus string

const (
	HealthGreen  HealthStatus = "green"
	HealthYellow HealthStatus = "yellow"
	HealthRed    HealthStatus = "red"
)

type Observation struct {
	SessionID string
	Alive     bool
	LogMTime  time.Time
	LogSize   int64
}

type SessionClaims struct {
	SessionID    string    `json:"session_id"`
	HasEvents    bool      `json:"has_events"`
	Completed    bool      `json:"completed"`
	Provider     string    `json:"provider,omitempty"`
	Model        string    `json:"model,omitempty"`
	TotalCostUSD float64   `json:"total_cost_usd"`
	FirstEventAt time.Time `json:"first_event_at"`
	LastEventAt  time.Time `json:"last_event_at"`
	LastAction   string    `json:"last_action,omitempty"`
	TokensIn     int       `json:"tokens_in"`
	TokensOut    int       `json:"tokens_out"`
	Failed       bool      `json:"failed"`
}

type SessionMeta struct {
	SessionID               string        `json:"session_id"`
	Status                  SessionStatus `json:"status"`
	Provider                string        `json:"provider,omitempty"`
	Model                   string        `json:"model,omitempty"`
	TotalCostUSD            float64       `json:"total_cost_usd"`
	DurationSeconds         int64         `json:"duration_seconds"`
	LastActivity            time.Time     `json:"last_activity"`
	CurrentAction           string        `json:"current_action,omitempty"`
	Health                  HealthStatus  `json:"health"`
	ContextWindowUsagePct   float64       `json:"context_window_usage_pct"`
	RetryCount              int           `json:"retry_count"`
	Alive                   bool          `json:"alive"`
	Stuck                   bool          `json:"stuck"`
	LogSize                 int64         `json:"log_size"`
	UpdatedAt               time.Time     `json:"updated_at"`
	IdleSeconds             int64         `json:"idle_seconds"`
	StuckThresholdSeconds   int64         `json:"stuck_threshold_seconds"`
	LastObservedProviderRaw string        `json:"last_observed_provider_raw,omitempty"`
}

type Observer interface {
	Observe(sessionID string) (Observation, error)
}

type ClaimsReader interface {
	ReadSession(sessionID string) (SessionClaims, error)
}

type TicketMaterializer interface {
	Materialize(ctx context.Context, sessionIDs []string) error
}

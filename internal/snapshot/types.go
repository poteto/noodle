package snapshot

//go:generate tygo generate --config ../../tygo.yaml

import (
	"encoding/json"
	"time"

	"github.com/poteto/noodle/loop"
)

// TraceFilter categorizes event lines for filtering.
type TraceFilter string

const (
	TraceFilterAll    TraceFilter = "all"
	TraceFilterTools  TraceFilter = "tools"
	TraceFilterThink  TraceFilter = "think"
	TraceFilterTicket TraceFilter = "ticket"
)

// LoopState constants.
const (
	LoopStateRunning  = "running"
	LoopStatePaused   = "paused"
	LoopStateDraining = "draining"
	LoopStateIdle     = "idle"
)

// Health constants.
const (
	HealthGreen  = "green"
	HealthYellow = "yellow"
	HealthRed    = "red"
)

// Snapshot is a point-in-time view of all loop state.
type Snapshot struct {
	UpdatedAt time.Time `json:"updated_at"`
	LoopState string    `json:"loop_state"`

	Sessions []Session `json:"sessions"`
	Active   []Session `json:"active"`
	Recent   []Session `json:"recent"`
	Orders   []Order   `json:"orders"`

	ActiveOrderIDs  []string                `json:"active_order_ids"`
	ActionNeeded    []string                `json:"action_needed"`
	EventsBySession map[string][]EventLine  `json:"events_by_session"`
	FeedEvents      []FeedEvent             `json:"feed_events"`
	TotalCostUSD    float64                 `json:"total_cost_usd"`

	PendingReviews     []loop.PendingReviewItem `json:"pending_reviews"`
	PendingReviewCount int                      `json:"pending_review_count"`
	Autonomy           string                   `json:"autonomy"`
	MaxCooks           int                      `json:"max_cooks"`
	Warnings           []string                 `json:"warnings"`
}

// Session represents a single agent session.
type Session struct {
	ID                    string    `json:"id"`
	DisplayName           string    `json:"display_name"`
	Status                string    `json:"status"`
	Runtime               string    `json:"runtime"`
	Provider              string    `json:"provider"`
	Model                 string    `json:"model"`
	TotalCostUSD          float64   `json:"total_cost_usd"`
	DurationSeconds       int64     `json:"duration_seconds"`
	LastActivity          time.Time `json:"last_activity"`
	CurrentAction         string    `json:"current_action"`
	Health                string    `json:"health"`
	ContextWindowUsagePct float64   `json:"context_window_usage_pct"`
	RetryCount            int       `json:"retry_count"`
	IdleSeconds           int64     `json:"idle_seconds"`
	StuckThresholdSeconds int64     `json:"stuck_threshold_seconds"`
	LoopState             string    `json:"loop_state"`
	RemoteHost            string    `json:"remote_host,omitempty"`
	DispatchWarning       string    `json:"dispatch_warning,omitempty"`
	WorktreeName          string    `json:"worktree_name,omitempty"`
	TaskKey               string    `json:"task_key,omitempty"`
	Title                 string    `json:"title,omitempty"`
}

// Order is a pipeline of stages in the snapshot.
type Order struct {
	ID        string   `json:"id"`
	Title     string   `json:"title,omitempty"`
	Plan      []string `json:"plan,omitempty"`
	Rationale string   `json:"rationale,omitempty"`
	Stages    []Stage  `json:"stages"`
	Status    string   `json:"status"`
}

// Stage is a unit of work within an order in the snapshot.
type Stage struct {
	TaskKey   string                     `json:"task_key,omitempty"`
	Prompt    string                     `json:"prompt,omitempty"`
	Skill     string                     `json:"skill,omitempty"`
	Provider  string                     `json:"provider"`
	Model     string                     `json:"model"`
	Runtime   string                     `json:"runtime,omitempty"`
	Group     int                        `json:"group,omitempty"`
	Status    string                     `json:"status"`
	Extra     map[string]json.RawMessage `json:"extra,omitempty"`
	SessionID string                     `json:"session_id,omitempty"`
}

// EventLine is a single parsed event in a session trace.
type EventLine struct {
	At       time.Time   `json:"at"`
	Label    string      `json:"label"`
	Body     string      `json:"body"`
	Category TraceFilter `json:"category"`
}

// FeedEvent is one event in the feed timeline.
type FeedEvent struct {
	SessionID string    `json:"session_id"`
	AgentName string    `json:"agent_name"`
	TaskType  string    `json:"task_type"`
	At        time.Time `json:"at"`
	Label     string    `json:"label"`
	Body      string    `json:"body"`
	Category  string    `json:"category"`
}

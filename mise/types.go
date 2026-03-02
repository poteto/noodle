package mise

import (
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/event"
)

type ActiveSummary struct {
	Total     int            `json:"total"`
	ByTaskKey map[string]int `json:"by_task_key,omitempty"`
	ByStatus  map[string]int `json:"by_status,omitempty"`
	ByRuntime map[string]int `json:"by_runtime,omitempty"`
}

type ResourceSnapshot struct {
	MaxConcurrency int `json:"max_concurrency"`
	Active         int `json:"active"`
	Available      int `json:"available"`
}

type HistoryItem struct {
	SessionID   string    `json:"session_id"`
	OrderID     string    `json:"order_id,omitempty"`
	TaskKey     string    `json:"task_key,omitempty"`
	Status      string    `json:"status"`
	DurationS   int64     `json:"duration_s"`
	CompletedAt time.Time `json:"completed_at"`
}

type RoutingSnapshot struct {
	Defaults          RoutingPolicy `json:"defaults"`
	AvailableRuntimes []string      `json:"available_runtimes,omitempty"`
}

type RoutingPolicy struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

// TaskTypeSummary describes a schedulable task type discovered from skill frontmatter.
type TaskTypeSummary struct {
	Key      string `json:"key"`
	Schedule string `json:"schedule"`
}

// RecentEvent is a lifecycle event emitted by the loop, surfaced in the brief
// so the schedule agent can react to state changes.
type RecentEvent struct {
	Type    string    `json:"type"`
	Seq     uint64    `json:"seq"`
	At      time.Time `json:"at"`
	Summary string    `json:"summary"`
}

type Brief struct {
	GeneratedAt   time.Time             `json:"generated_at"`
	Backlog       []adapter.BacklogItem `json:"backlog"`
	ActiveSummary ActiveSummary         `json:"active_summary"`
	Tickets       []event.Ticket        `json:"tickets"`
	Resources     ResourceSnapshot      `json:"resources"`
	RecentHistory []HistoryItem         `json:"recent_history"`
	RecentEvents  []RecentEvent         `json:"recent_events"`
	Routing       RoutingSnapshot       `json:"routing"`
	TaskTypes     []TaskTypeSummary     `json:"task_types,omitempty"`
	Warnings      []string              `json:"warnings,omitempty"`
}

package mise

import (
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/event"
)

// PlanSummary is the plan entry in the mise brief (replaces adapter.PlanItem).
type PlanSummary struct {
	ID        int            `json:"id"`
	Title     string         `json:"title"`
	Status    string         `json:"status"`
	Provider  string         `json:"provider,omitempty"`
	Model     string         `json:"model,omitempty"`
	Backlog   string         `json:"backlog,omitempty"`
	Directory string         `json:"directory"` // slug
	Phases    []PhaseSummary `json:"phases"`
}

// PhaseSummary describes one phase within a plan summary.
type PhaseSummary struct {
	Name     string `json:"name"`
	Filename string `json:"filename"`
	Status   string `json:"status"`
}

type ActiveCook struct {
	ID        string  `json:"id"`
	Target    string  `json:"target,omitempty"`
	Provider  string  `json:"provider,omitempty"`
	Model     string  `json:"model,omitempty"`
	Status    string  `json:"status"`
	Cost      float64 `json:"cost"`
	DurationS int64   `json:"duration_s"`
}

type ResourceSnapshot struct {
	MaxCooks  int `json:"max_cooks"`
	Active    int `json:"active"`
	Available int `json:"available"`
}

type HistoryItem struct {
	Target    string  `json:"target,omitempty"`
	CookID    string  `json:"cook_id"`
	Outcome   string  `json:"outcome"`
	Cost      float64 `json:"cost"`
	DurationS int64   `json:"duration_s"`
}

type RoutingSnapshot struct {
	Defaults          RoutingPolicy            `json:"defaults"`
	Tags              map[string]RoutingPolicy `json:"tags"`
	AvailableRuntimes []string                 `json:"available_runtimes,omitempty"`
}

type RoutingPolicy struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

// TaskTypeSummary describes a schedulable task type discovered from skill frontmatter.
type TaskTypeSummary struct {
	Key      string `json:"key"`
	CanMerge bool   `json:"can_merge"`
	Schedule string `json:"schedule"`
}

type Brief struct {
	GeneratedAt     time.Time             `json:"generated_at"`
	Backlog         []adapter.BacklogItem `json:"backlog"`
	Plans           []PlanSummary         `json:"plans"`
	NeedsScheduling []int                 `json:"needs_scheduling,omitempty"`
	ActiveCooks     []ActiveCook          `json:"active_cooks"`
	Tickets         []event.Ticket        `json:"tickets"`
	Resources       ResourceSnapshot      `json:"resources"`
	RecentHistory   []HistoryItem         `json:"recent_history"`
	Routing         RoutingSnapshot       `json:"routing"`
	TaskTypes       []TaskTypeSummary     `json:"task_types,omitempty"`
	QualityVerdicts []QualityVerdict      `json:"quality_verdicts,omitempty"`
	Warnings        []string              `json:"warnings,omitempty"`
}

package mise

import (
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/event"
)

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
	Defaults RoutingPolicy            `json:"defaults"`
	Tags     map[string]RoutingPolicy `json:"tags"`
}

type RoutingPolicy struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type Brief struct {
	GeneratedAt    time.Time           `json:"generated_at"`
	Backlog        []adapter.BacklogItem `json:"backlog"`
	Plans          []adapter.PlanItem  `json:"plans"`
	ActiveCooks    []ActiveCook        `json:"active_cooks"`
	Tickets        []event.Ticket      `json:"tickets"`
	Resources      ResourceSnapshot    `json:"resources"`
	RecentHistory  []HistoryItem       `json:"recent_history"`
	Routing        RoutingSnapshot     `json:"routing"`
	Warnings       []string            `json:"warnings,omitempty"`
}

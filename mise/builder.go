package mise

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/event"
)

type Builder struct {
	projectDir string
	runtimeDir string
	config     config.Config
	runner     *adapter.Runner
	now        func() time.Time
	TaskTypes  []TaskTypeSummary
}

func NewBuilder(projectDir string, cfg config.Config) *Builder {
	projectDir = strings.TrimSpace(projectDir)
	return &Builder{
		projectDir: projectDir,
		runtimeDir: filepath.Join(projectDir, ".noodle"),
		config:     cfg,
		runner:     adapter.NewRunner(projectDir, cfg),
		now:        time.Now,
	}
}

func (b *Builder) Build(ctx context.Context) (Brief, []string, error) {
	warnings := make([]string, 0)
	backlog := make([]adapter.BacklogItem, 0)
	plans := make([]adapter.PlanItem, 0)

	if _, ok := b.config.Adapters["backlog"]; ok {
		if strings.TrimSpace(b.config.Adapters["backlog"].Scripts["sync"]) == "" {
			warnings = append(warnings, "backlog sync script missing; returning empty backlog")
		} else {
			items, err := b.runner.SyncBacklog(ctx)
			if err != nil {
				if isMissingSyncScriptError(err) {
					warnings = append(warnings, "backlog sync script missing; returning empty backlog")
				} else {
					return Brief{}, warnings, err
				}
			} else {
				backlog = filterActiveBacklog(items)
			}
		}
	}

	if _, ok := b.config.Adapters["plans"]; ok {
		if strings.TrimSpace(b.config.Adapters["plans"].Scripts["sync"]) == "" {
			warnings = append(warnings, "plans sync script missing; returning empty plans")
		} else {
			items, err := b.runner.SyncPlans(ctx)
			if err != nil {
				if isMissingSyncScriptError(err) {
					warnings = append(warnings, "plans sync script missing; returning empty plans")
				} else {
					return Brief{}, warnings, err
				}
			} else {
				plans = filterActivePlans(items)
			}
		}
	}

	activeCooks, recentHistory, err := b.readSessionState()
	if err != nil {
		return Brief{}, warnings, err
	}
	tickets, err := b.readTickets()
	if err != nil {
		return Brief{}, warnings, err
	}

	resources := ResourceSnapshot{
		MaxCooks: b.config.Concurrency.MaxCooks,
		Active:   len(activeCooks),
	}
	resources.Available = resources.MaxCooks - resources.Active
	if resources.Available < 0 {
		resources.Available = 0
	}

	routing := RoutingSnapshot{
		Defaults: RoutingPolicy{
			Provider: b.config.Routing.Defaults.Provider,
			Model:    b.config.Routing.Defaults.Model,
		},
		Tags: make(map[string]RoutingPolicy, len(b.config.Routing.Tags)),
	}
	for tag, policy := range b.config.Routing.Tags {
		routing.Tags[tag] = RoutingPolicy{Provider: policy.Provider, Model: policy.Model}
	}

	brief := Brief{
		GeneratedAt:   b.now().UTC(),
		Backlog:       backlog,
		Plans:         plans,
		ActiveCooks:   activeCooks,
		Tickets:       tickets,
		Resources:     resources,
		RecentHistory: recentHistory,
		Routing:       routing,
		TaskTypes:     b.TaskTypes,
		Warnings:      warnings,
	}

	if err := writeBriefAtomic(filepath.Join(b.runtimeDir, "mise.json"), brief); err != nil {
		return Brief{}, warnings, err
	}
	return brief, warnings, nil
}

func filterActiveBacklog(items []adapter.BacklogItem) []adapter.BacklogItem {
	filtered := make([]adapter.BacklogItem, 0, len(items))
	for _, item := range items {
		if item.Status == adapter.BacklogStatusDone {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func filterActivePlans(items []adapter.PlanItem) []adapter.PlanItem {
	filtered := make([]adapter.PlanItem, 0, len(items))
	for _, item := range items {
		if item.Status == adapter.PlanStatusDone {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func (b *Builder) readSessionState() ([]ActiveCook, []HistoryItem, error) {
	sessionsPath := filepath.Join(b.runtimeDir, "sessions")
	entries, err := os.ReadDir(sessionsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("read sessions directory: %w", err)
	}

	active := make([]ActiveCook, 0, len(entries))
	history := make([]struct {
		item      HistoryItem
		updatedAt time.Time
	}, 0, len(entries))

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaPath := filepath.Join(sessionsPath, entry.Name(), "meta.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, nil, fmt.Errorf("read session meta %s: %w", entry.Name(), err)
		}

		var meta struct {
			SessionID       string    `json:"session_id"`
			Status          string    `json:"status"`
			Provider        string    `json:"provider"`
			Model           string    `json:"model"`
			TotalCostUSD    float64   `json:"total_cost_usd"`
			DurationSeconds int64     `json:"duration_seconds"`
			UpdatedAt       time.Time `json:"updated_at"`
		}
		if err := json.Unmarshal(data, &meta); err != nil {
			return nil, nil, fmt.Errorf("parse session meta %s: %w", entry.Name(), err)
		}

		cookID := strings.TrimSpace(meta.SessionID)
		if cookID == "" {
			cookID = entry.Name()
		}

		switch meta.Status {
		case "running", "stuck", "spawning":
			active = append(active, ActiveCook{
				ID:        cookID,
				Provider:  meta.Provider,
				Model:     meta.Model,
				Status:    meta.Status,
				Cost:      meta.TotalCostUSD,
				DurationS: meta.DurationSeconds,
			})
		case "exited", "failed", "completed":
			history = append(history, struct {
				item      HistoryItem
				updatedAt time.Time
			}{
				item: HistoryItem{
					CookID:    cookID,
					Outcome:   meta.Status,
					Cost:      meta.TotalCostUSD,
					DurationS: meta.DurationSeconds,
				},
				updatedAt: meta.UpdatedAt,
			})
		}
	}

	sort.SliceStable(active, func(i, j int) bool {
		return active[i].ID < active[j].ID
	})
	sort.SliceStable(history, func(i, j int) bool {
		if history[i].updatedAt.Equal(history[j].updatedAt) {
			return history[i].item.CookID < history[j].item.CookID
		}
		return history[i].updatedAt.After(history[j].updatedAt)
	})

	recent := make([]HistoryItem, 0, len(history))
	for _, item := range history {
		recent = append(recent, item.item)
		if len(recent) >= 20 {
			break
		}
	}
	return active, recent, nil
}

func (b *Builder) readTickets() ([]event.Ticket, error) {
	path := filepath.Join(b.runtimeDir, "tickets.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []event.Ticket{}, nil
		}
		return nil, fmt.Errorf("read tickets.json: %w", err)
	}
	var tickets []event.Ticket
	if err := json.Unmarshal(data, &tickets); err != nil {
		return nil, fmt.Errorf("parse tickets.json: %w", err)
	}
	if tickets == nil {
		return []event.Ticket{}, nil
	}
	return tickets, nil
}

func isMissingSyncScriptError(err error) bool {
	raw := strings.ToLower(err.Error())
	switch {
	case strings.Contains(raw, "script is not configured"):
		return true
	case strings.Contains(raw, " not found"):
		return true
	case strings.Contains(raw, "no such file"):
		return true
	default:
		return false
	}
}

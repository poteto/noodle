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
	"github.com/poteto/noodle/internal/sessionmeta"
	"github.com/poteto/noodle/plan"
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
	plans := make([]PlanSummary, 0)

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

	// Plans: native reader (replaces adapter sync)
	plansDir := filepath.Join(b.projectDir, "brain", "plans")
	nativePlans, planErr := plan.ReadAll(plansDir)
	if planErr != nil {
		warnings = append(warnings, "plan reading failed: "+planErr.Error())
	} else {
		plans = toPlanSummaries(nativePlans)
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

	// Quality verdicts
	verdicts, verdictErr := ReadQualityVerdicts(b.runtimeDir)
	if verdictErr != nil {
		warnings = append(warnings, "quality verdict reading failed: "+verdictErr.Error())
	}

	brief := Brief{
		GeneratedAt:     b.now().UTC(),
		Backlog:         backlog,
		Plans:           plans,
		ActiveCooks:     activeCooks,
		Tickets:         tickets,
		Resources:       resources,
		RecentHistory:   recentHistory,
		Routing:         routing,
		TaskTypes:       b.TaskTypes,
		QualityVerdicts: verdicts,
		Warnings:        warnings,
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

func toPlanSummaries(plans []plan.Plan) []PlanSummary {
	summaries := make([]PlanSummary, 0, len(plans))
	for _, p := range plans {
		phases := make([]PhaseSummary, 0, len(p.Phases))
		for _, ph := range p.Phases {
			phases = append(phases, PhaseSummary{
				Name:     ph.Name,
				Filename: ph.Filename,
				Status:   ph.Status,
			})
		}
		summaries = append(summaries, PlanSummary{
			ID:        p.Meta.ID,
			Title:     p.Title,
			Status:    p.Meta.Status,
			Provider:  p.Meta.Provider,
			Model:     p.Meta.Model,
			Directory: p.Slug,
			Phases:    phases,
		})
	}
	return summaries
}

func (b *Builder) readSessionState() ([]ActiveCook, []HistoryItem, error) {
	metas, err := sessionmeta.ReadAll(b.runtimeDir)
	if err != nil {
		return nil, nil, err
	}
	if len(metas) == 0 {
		return nil, nil, nil
	}

	active := make([]ActiveCook, 0, len(metas))
	history := make([]struct {
		item      HistoryItem
		updatedAt time.Time
	}, 0, len(metas))

	for _, meta := range metas {
		cookID := strings.TrimSpace(meta.SessionID)
		if cookID == "" {
			continue
		}
		status := strings.ToLower(strings.TrimSpace(meta.Status))
		switch status {
		case "running", "stuck", "spawning":
			active = append(active, ActiveCook{
				ID:        cookID,
				Provider:  strings.TrimSpace(meta.Provider),
				Model:     strings.TrimSpace(meta.Model),
				Status:    status,
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
					Outcome:   status,
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

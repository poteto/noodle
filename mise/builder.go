package mise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/plan"
)

type Builder struct {
	projectDir  string
	runtimeDir  string
	config      config.Config
	runner      *adapter.Runner
	now         func() time.Time
	TaskTypes   []TaskTypeSummary
	lastContent []byte // JSON sans GeneratedAt for change detection
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

func (b *Builder) Build(ctx context.Context, activeSummary ActiveSummary, recentHistory []HistoryItem) (Brief, []string, error) {
	warnings := make([]string, 0)
	backlog := make([]adapter.BacklogItem, 0)
	plans := make([]PlanSummary, 0)
	nativePlans := make([]plan.Plan, 0)

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
	readPlans, planErr := plan.ReadAll(plansDir)
	if planErr != nil {
		warnings = append(warnings, "plan reading failed: "+planErr.Error())
	} else {
		nativePlans = readPlans
		plans = toPlanSummaries(nativePlans)
	}

	tickets, err := b.readTickets()
	if err != nil {
		return Brief{}, warnings, err
	}

	if activeSummary.ByTaskKey == nil {
		activeSummary.ByTaskKey = map[string]int{}
	}
	if activeSummary.ByStatus == nil {
		activeSummary.ByStatus = map[string]int{}
	}
	if activeSummary.ByRuntime == nil {
		activeSummary.ByRuntime = map[string]int{}
	}
	if recentHistory == nil {
		recentHistory = []HistoryItem{}
	}

	resources := ResourceSnapshot{
		MaxCooks: b.config.Concurrency.MaxCooks,
		Active:   activeSummary.Total,
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
		Tags:              make(map[string]RoutingPolicy, len(b.config.Routing.Tags)),
		AvailableRuntimes: b.config.AvailableRuntimes(),
	}
	for tag, policy := range b.config.Routing.Tags {
		routing.Tags[tag] = RoutingPolicy{Provider: policy.Provider, Model: policy.Model}
	}

	brief := Brief{
		GeneratedAt:   b.now().UTC(),
		Backlog:       backlog,
		Plans:         plans,
		ActiveSummary: activeSummary,
		Tickets:       tickets,
		Resources:     resources,
		RecentHistory: recentHistory,
		Routing:       routing,
		TaskTypes:     b.TaskTypes,
		Warnings:      warnings,
	}

	// Only write when content actually changed (ignore GeneratedAt).
	cmp := brief
	cmp.GeneratedAt = time.Time{}
	content, err := json.Marshal(cmp)
	if err != nil {
		return Brief{}, warnings, fmt.Errorf("encode mise json for comparison: %w", err)
	}
	if !bytes.Equal(content, b.lastContent) {
		if err := writeBriefAtomic(filepath.Join(b.runtimeDir, "mise.json"), brief); err != nil {
			return Brief{}, warnings, err
		}
		b.lastContent = content
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
			Backlog:   p.Meta.Backlog,
			Directory: p.Slug,
			Phases:    phases,
		})
	}
	return summaries
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

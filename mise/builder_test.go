package mise

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/plan"
)

func TestBuilderBuildWritesMiseJSON(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".noodle", "sessions", "cook-a"), 0o755); err != nil {
		t.Fatalf("mkdir runtime sessions: %v", err)
	}

	meta := map[string]any{
		"session_id":               "cook-a",
		"status":                   "running",
		"provider":                 "claude",
		"model":                    "claude-sonnet-4-6",
		"total_cost_usd":           0.22,
		"duration_seconds":         80,
		"updated_at":               "2026-02-22T16:00:00Z",
		"current_action":           "write tests",
		"last_activity":            "2026-02-22T16:00:00Z",
		"idle_seconds":             0,
		"context_window_usage_pct": 4.0,
	}
	metaBytes, _ := json.Marshal(meta)
	if err := os.WriteFile(filepath.Join(projectDir, ".noodle", "sessions", "cook-a", "meta.json"), metaBytes, 0o644); err != nil {
		t.Fatalf("write session meta: %v", err)
	}

	tickets := []event.Ticket{{
		Target:     "42",
		TargetType: event.TicketTargetBacklogItem,
		CookID:     "cook-a",
		Status:     event.TicketStatusActive,
		ClaimedAt:  time.Date(2026, 2, 22, 16, 0, 0, 0, time.UTC),
	}}
	ticketBytes, _ := json.Marshal(tickets)
	if err := os.WriteFile(filepath.Join(projectDir, ".noodle", "tickets.json"), ticketBytes, 0o644); err != nil {
		t.Fatalf("write tickets: %v", err)
	}

	// Set up native plan files in brain/plans/
	plansDir := filepath.Join(projectDir, "brain", "plans", "03-auth-refactor")
	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		t.Fatalf("mkdir plans: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "brain", "plans", "index.md"),
		[]byte("# Plans\n\n- [[plans/03-auth-refactor/overview]]\n"), 0o644); err != nil {
		t.Fatalf("write plans index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(plansDir, "overview.md"),
		[]byte("---\nid: 3\ncreated: 2026-02-20\nstatus: active\nprovider: codex\nmodel: gpt-5.3-codex\n---\n\n# Auth Refactor\n\n- [ ] [[plans/03-auth-refactor/phase-01-implement]] -- Implement\n"), 0o644); err != nil {
		t.Fatalf("write plan overview: %v", err)
	}
	if err := os.WriteFile(filepath.Join(plansDir, "phase-01-implement.md"),
		[]byte("# Implement\n\nPhase details.\n"), 0o644); err != nil {
		t.Fatalf("write plan phase: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Adapters = map[string]config.AdapterConfig{
		"backlog": {
			Scripts: map[string]string{
				"sync": "printf '%s\\n' '{\"id\":\"42\",\"title\":\"Fix login\",\"status\":\"open\"}' '{\"id\":\"43\",\"title\":\"Done item\",\"status\":\"done\"}'",
			},
		},
	}

	builder := NewBuilder(projectDir, cfg)
	builder.now = func() time.Time { return time.Date(2026, 2, 22, 16, 1, 0, 0, time.UTC) }

	brief, warnings, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("build mise: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v", warnings)
	}
	if len(brief.Backlog) != 1 {
		t.Fatalf("backlog count = %d", len(brief.Backlog))
	}
	if brief.Backlog[0].ID != "42" {
		t.Fatalf("backlog id = %q", brief.Backlog[0].ID)
	}
	if len(brief.Plans) != 1 {
		t.Fatalf("plans count = %d, want 1", len(brief.Plans))
	}
	if brief.Plans[0].ID != 3 {
		t.Fatalf("plan id = %d, want 3", brief.Plans[0].ID)
	}
	if brief.Plans[0].Title != "Auth Refactor" {
		t.Fatalf("plan title = %q, want %q", brief.Plans[0].Title, "Auth Refactor")
	}
	if brief.Plans[0].Status != "active" {
		t.Fatalf("plan status = %q, want %q", brief.Plans[0].Status, "active")
	}
	if brief.Plans[0].Provider != "codex" {
		t.Fatalf("plan provider = %q, want %q", brief.Plans[0].Provider, "codex")
	}
	if brief.Plans[0].Model != "gpt-5.3-codex" {
		t.Fatalf("plan model = %q, want %q", brief.Plans[0].Model, "gpt-5.3-codex")
	}
	if len(brief.Plans[0].Phases) != 1 {
		t.Fatalf("plan phase count = %d, want 1", len(brief.Plans[0].Phases))
	}
	if brief.Plans[0].Phases[0].Status != "active" {
		t.Fatalf("phase status = %q, want %q", brief.Plans[0].Phases[0].Status, "active")
	}
	if len(brief.ActiveCooks) != 1 {
		t.Fatalf("active cooks = %d", len(brief.ActiveCooks))
	}
	if len(brief.Tickets) != 1 {
		t.Fatalf("tickets count = %d", len(brief.Tickets))
	}

	misePath := filepath.Join(projectDir, ".noodle", "mise.json")
	if _, err := os.Stat(misePath); err != nil {
		t.Fatalf("mise.json not written: %v", err)
	}
}

func TestSchedulablePlanIDsFallsBackToPlanID(t *testing.T) {
	tests := []struct {
		name    string
		plans   []plan.Plan
		backlog []adapter.BacklogItem
		want    []int
	}{
		{
			name: "explicit backlog field",
			plans: []plan.Plan{
				{Meta: plan.PlanMeta{ID: 10, Status: "active", Backlog: "42"}},
			},
			backlog: []adapter.BacklogItem{
				{ID: "42", Status: adapter.BacklogStatusOpen},
			},
			want: []int{10},
		},
		{
			name: "implicit match via plan ID",
			plans: []plan.Plan{
				{Meta: plan.PlanMeta{ID: 38, Status: "active"}},
			},
			backlog: []adapter.BacklogItem{
				{ID: "38", Status: adapter.BacklogStatusOpen},
			},
			want: []int{38},
		},
		{
			name: "explicit field takes precedence over plan ID",
			plans: []plan.Plan{
				{Meta: plan.PlanMeta{ID: 38, Status: "active", Backlog: "99"}},
			},
			backlog: []adapter.BacklogItem{
				{ID: "38", Status: adapter.BacklogStatusOpen},
				{ID: "99", Status: adapter.BacklogStatusOpen},
			},
			want: []int{38},
		},
		{
			name: "no match when backlog item is done",
			plans: []plan.Plan{
				{Meta: plan.PlanMeta{ID: 38, Status: "active"}},
			},
			backlog: []adapter.BacklogItem{
				{ID: "38", Status: adapter.BacklogStatusDone},
			},
			want: nil,
		},
		{
			name: "no match when no backlog items",
			plans: []plan.Plan{
				{Meta: plan.PlanMeta{ID: 38, Status: "active"}},
			},
			backlog: nil,
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := schedulablePlanIDs(tt.plans, tt.backlog)
			if len(got) != len(tt.want) {
				t.Fatalf("schedulablePlanIDs = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("schedulablePlanIDs[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestBuilderMissingSyncScriptWarnsAndContinues(t *testing.T) {
	projectDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Adapters = map[string]config.AdapterConfig{
		"backlog": {
			Scripts: map[string]string{
				"sync": ".noodle/adapters/backlog-sync",
			},
		},
	}

	builder := NewBuilder(projectDir, cfg)
	brief, warnings, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("build mise: %v", err)
	}
	if len(brief.Backlog) != 0 {
		t.Fatalf("expected empty backlog, got %d items", len(brief.Backlog))
	}
	if len(warnings) == 0 {
		t.Fatal("expected warning for missing sync script")
	}
	if !strings.Contains(strings.ToLower(strings.Join(warnings, "\n")), "missing") {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
}

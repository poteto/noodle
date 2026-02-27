package mise

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/event"
)

func TestBuilderBuildWritesMiseJSON(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".noodle"), 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
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

	brief, warnings, err := builder.Build(context.Background(), ActiveSummary{
		Total:     1,
		ByTaskKey: map[string]int{"execute": 1},
		ByStatus:  map[string]int{"active": 1},
		ByRuntime: map[string]int{"tmux": 1},
	}, []HistoryItem{{
		SessionID:   "cook-a",
		OrderID:     "42",
		TaskKey:     "execute",
		Status:      "completed",
		DurationS:   80,
		CompletedAt: time.Date(2026, 2, 22, 16, 0, 0, 0, time.UTC),
	}})
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
	if brief.ActiveSummary.Total != 1 {
		t.Fatalf("active summary total = %d", brief.ActiveSummary.Total)
	}
	if len(brief.Tickets) != 1 {
		t.Fatalf("tickets count = %d", len(brief.Tickets))
	}

	misePath := filepath.Join(projectDir, ".noodle", "mise.json")
	if _, err := os.Stat(misePath); err != nil {
		t.Fatalf("mise.json not written: %v", err)
	}

	// Second build with identical inputs should skip the write.
	firstInfo, _ := os.Stat(misePath)
	firstMod := firstInfo.ModTime()

	// Advance the clock so GeneratedAt differs, but content is the same.
	builder.now = func() time.Time { return time.Date(2026, 2, 22, 16, 2, 0, 0, time.UTC) }
	_, _, err = builder.Build(context.Background(), ActiveSummary{
		Total:     1,
		ByTaskKey: map[string]int{"execute": 1},
		ByStatus:  map[string]int{"active": 1},
		ByRuntime: map[string]int{"tmux": 1},
	}, []HistoryItem{{
		SessionID:   "cook-a",
		OrderID:     "42",
		TaskKey:     "execute",
		Status:      "completed",
		DurationS:   80,
		CompletedAt: time.Date(2026, 2, 22, 16, 0, 0, 0, time.UTC),
	}})
	if err != nil {
		t.Fatalf("second build: %v", err)
	}

	secondInfo, _ := os.Stat(misePath)
	if secondInfo.ModTime() != firstMod {
		t.Fatal("mise.json rewritten despite no content change")
	}
}

func TestAllPlansAppearRegardlessOfBacklogStatus(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".noodle"), 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	// Create two plans: one with a done backlog item, one with no backlog link.
	plansDir := filepath.Join(projectDir, "brain", "plans")
	for _, dir := range []string{"10-done-backlog", "20-no-backlog"} {
		if err := os.MkdirAll(filepath.Join(plansDir, dir), 0o755); err != nil {
			t.Fatalf("mkdir plan: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(plansDir, "index.md"),
		[]byte("# Plans\n\n- [[plans/10-done-backlog/overview]]\n- [[plans/20-no-backlog/overview]]\n"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(plansDir, "10-done-backlog", "overview.md"),
		[]byte("---\nid: 10\ncreated: 2026-02-20\nstatus: active\nbacklog: \"42\"\n---\n\n# Done Backlog Plan\n"), 0o644); err != nil {
		t.Fatalf("write plan 10: %v", err)
	}
	if err := os.WriteFile(filepath.Join(plansDir, "20-no-backlog", "overview.md"),
		[]byte("---\nid: 20\ncreated: 2026-02-20\nstatus: active\n---\n\n# No Backlog Plan\n"), 0o644); err != nil {
		t.Fatalf("write plan 20: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Adapters = map[string]config.AdapterConfig{
		"backlog": {
			Scripts: map[string]string{
				"sync": "printf '%s\\n' '{\"id\":\"42\",\"title\":\"Done item\",\"status\":\"done\"}'",
			},
		},
	}

	builder := NewBuilder(projectDir, cfg)
	builder.now = func() time.Time { return time.Date(2026, 2, 22, 16, 1, 0, 0, time.UTC) }

	brief, _, err := builder.Build(context.Background(), ActiveSummary{}, nil)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if len(brief.Plans) != 2 {
		t.Fatalf("plans count = %d, want 2 (all plans regardless of backlog status)", len(brief.Plans))
	}

	ids := map[int]bool{}
	for _, p := range brief.Plans {
		ids[p.ID] = true
	}
	if !ids[10] {
		t.Fatal("plan 10 (done backlog) missing from brief.Plans")
	}
	if !ids[20] {
		t.Fatal("plan 20 (no backlog) missing from brief.Plans")
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
	brief, warnings, err := builder.Build(context.Background(), ActiveSummary{}, nil)
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

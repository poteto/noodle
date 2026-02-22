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
	if err := os.MkdirAll(filepath.Join(projectDir, ".noodle", "sessions", "cook-a"), 0o755); err != nil {
		t.Fatalf("mkdir runtime sessions: %v", err)
	}

	meta := map[string]any{
		"session_id":        "cook-a",
		"status":            "running",
		"provider":          "claude",
		"model":             "claude-sonnet-4-6",
		"total_cost_usd":    0.22,
		"duration_seconds":  80,
		"updated_at":        "2026-02-22T16:00:00Z",
		"current_action":    "write tests",
		"last_activity":     "2026-02-22T16:00:00Z",
		"idle_seconds":      0,
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

	cfg := config.DefaultConfig()
	cfg.Adapters = map[string]config.AdapterConfig{
		"backlog": {
			Scripts: map[string]string{
				"sync": "printf '%s\\n' '{\"id\":\"42\",\"title\":\"Fix login\",\"status\":\"open\"}' '{\"id\":\"43\",\"title\":\"Done item\",\"status\":\"done\"}'",
			},
		},
		"plans": {
			Scripts: map[string]string{
				"sync": "printf '%s\\n' '{\"id\":\"03\",\"title\":\"Auth refactor\",\"status\":\"active\",\"phases\":[{\"name\":\"implement\",\"status\":\"active\"}]}'",
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
		t.Fatalf("plans count = %d", len(brief.Plans))
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

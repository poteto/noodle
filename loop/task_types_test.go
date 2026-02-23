package loop

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
)

func TestConfiguredTaskTypesApplyConfigOverrides(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Adapters["plans"] = config.AdapterConfig{Skill: "plan-custom"}
	cfg.Adapters["backlog"] = config.AdapterConfig{Skill: "execute-custom"}
	cfg.SousChef.Skill = "priority-chef"
	cfg.Phases["oops"] = "oops-custom"
	cfg.Phases["debugging"] = "repair-custom"

	if got := planTaskSkill(cfg); got != "plan-custom" {
		t.Fatalf("plan task skill = %q, want plan-custom", got)
	}
	if got := executeTaskSkill(cfg); got != "execute-custom" {
		t.Fatalf("execute task skill = %q, want execute-custom", got)
	}
	if got := sousChefTaskSkill(cfg); got != "priority-chef" {
		t.Fatalf("sous-chef task skill = %q, want priority-chef", got)
	}
	if got := oopsTaskSkill(cfg); got != "oops-custom" {
		t.Fatalf("oops task skill = %q, want oops-custom", got)
	}
	if got := repairTaskSkill(cfg); got != "repair-custom" {
		t.Fatalf("repair task skill = %q, want repair-custom", got)
	}
}

func TestIsBlockingQueueItemReviewSkill(t *testing.T) {
	cfg := config.DefaultConfig()
	item := QueueItem{
		ID:    "review-42",
		Title: "Review after plan",
		Skill: "review",
	}
	if !isBlockingQueueItem(cfg, item) {
		t.Fatalf("expected review task to be blocking: %#v", item)
	}
}

func TestCycleBlocksConcurrentSpawnWhenBlockingTaskActive(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	queue := Queue{Items: []QueueItem{
		{ID: "review-1", Title: "Review plan", Skill: "review", Provider: "claude", Model: "claude-opus-4-6"},
		{ID: "exec-1", Title: "Implement feature", Skill: "execute", Provider: "codex", Model: "gpt-5.3-codex"},
	}}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Concurrency.MaxCooks = 2
	sp := &fakeSpawner{}
	l := New(projectDir, "noodle", cfg, Dependencies{
		Spawner:   sp,
		Worktree:  &fakeWorktree{},
		Adapter:   &fakeAdapterRunner{},
		Mise:      &fakeMise{},
		Monitor:   fakeMonitor{},
		Now:       time.Now,
		QueueFile: queuePath,
	})

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}
	if len(sp.calls) != 1 {
		t.Fatalf("spawn calls = %d, want 1", len(sp.calls))
	}
	if got := sp.calls[0].Name; got != "review-1-review-plan" {
		t.Fatalf("first spawn = %q, want review item", got)
	}
}

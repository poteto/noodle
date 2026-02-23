package loop

import (
	"testing"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
)

func TestApplyQueueRoutingDefaultsPreservesExplicitRouting(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Routing.Defaults.Provider = "codex"
	cfg.Routing.Defaults.Model = "gpt-5.3-codex"

	queue := Queue{
		Items: []QueueItem{
			{
				ID:       "plan-1",
				Provider: "claude",
				Model:    "claude-opus-4-6",
			},
		},
	}

	updated, changed := applyQueueRoutingDefaults(queue, cfg)
	if changed {
		t.Fatal("expected explicit routing to remain unchanged")
	}
	if got := updated.Items[0].Provider; got != "claude" {
		t.Fatalf("provider = %q, want claude", got)
	}
	if got := updated.Items[0].Model; got != "claude-opus-4-6" {
		t.Fatalf("model = %q, want claude-opus-4-6", got)
	}
}

func TestApplyQueueRoutingDefaultsFillsMissingRouting(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Routing.Defaults.Provider = "codex"
	cfg.Routing.Defaults.Model = "gpt-5.3-codex"
	cfg.Routing.Tags["plan"] = config.ModelPolicy{
		Provider: "claude",
		Model:    "claude-opus-4-6",
	}

	queue := Queue{
		Items: []QueueItem{
			{
				ID:       "task-1",
				TaskKey:  "plan",
				Provider: "",
				Model:    "",
			},
			{
				ID:       "task-2",
				Provider: "claude",
				Model:    "",
			},
		},
	}

	updated, changed := applyQueueRoutingDefaults(queue, cfg)
	if !changed {
		t.Fatal("expected missing routing fields to be defaulted")
	}
	if got := updated.Items[0].Provider; got != "claude" {
		t.Fatalf("task-1 provider = %q, want claude", got)
	}
	if got := updated.Items[0].Model; got != "claude-opus-4-6" {
		t.Fatalf("task-1 model = %q, want claude-opus-4-6", got)
	}
	if got := updated.Items[1].Provider; got != "claude" {
		t.Fatalf("task-2 provider = %q, want claude", got)
	}
	if got := updated.Items[1].Model; got != "gpt-5.3-codex" {
		t.Fatalf("task-2 model = %q, want gpt-5.3-codex", got)
	}
}

func TestNormalizeAndValidateQueueAssignsExecuteTaskKeyForBacklogItems(t *testing.T) {
	cfg := config.DefaultConfig()
	queue := Queue{
		Items: []QueueItem{
			{ID: "42", Title: "Implement fix"},
		},
	}
	backlog := []adapter.BacklogItem{
		{ID: "42", Status: adapter.BacklogStatusOpen},
	}

	updated, changed, err := normalizeAndValidateQueue(queue, backlog, cfg)
	if err != nil {
		t.Fatalf("normalizeAndValidateQueue error: %v", err)
	}
	if !changed {
		t.Fatal("expected queue normalization change")
	}
	if got := updated.Items[0].TaskKey; got != "execute" {
		t.Fatalf("task_key = %q, want execute", got)
	}
	if got := updated.Items[0].Skill; got != "backlog" {
		t.Fatalf("skill = %q, want backlog", got)
	}
}

func TestNormalizeAndValidateQueueRejectsUnknownSyntheticItem(t *testing.T) {
	cfg := config.DefaultConfig()
	queue := Queue{
		Items: []QueueItem{
			{ID: "synth-1", TaskKey: "new-type", Title: "brand new type"},
		},
	}

	_, _, err := normalizeAndValidateQueue(queue, nil, cfg)
	if err == nil {
		t.Fatal("expected validation error for unknown synthetic queue item")
	}
}

func TestNormalizeAndValidateQueueAllowsNonSyntheticWhenBacklogUnavailable(t *testing.T) {
	cfg := config.DefaultConfig()
	queue := Queue{
		Items: []QueueItem{
			{ID: "42", TaskKey: "execute", Title: "execute something"},
		},
	}

	_, _, err := normalizeAndValidateQueue(queue, nil, cfg)
	if err != nil {
		t.Fatalf("normalizeAndValidateQueue error: %v", err)
	}
}

func TestNormalizeAndValidateQueueRejectsNonSyntheticOutsideKnownBacklog(t *testing.T) {
	cfg := config.DefaultConfig()
	queue := Queue{
		Items: []QueueItem{
			{ID: "not-in-backlog", TaskKey: "execute", Title: "execute something"},
		},
	}
	backlog := []adapter.BacklogItem{
		{ID: "42", Status: adapter.BacklogStatusOpen},
	}

	_, _, err := normalizeAndValidateQueue(queue, backlog, cfg)
	if err == nil {
		t.Fatal("expected validation error for non-synthetic item not in known backlog")
	}
}

func TestNormalizeAndValidateQueueAllowsSyntheticReview(t *testing.T) {
	cfg := config.DefaultConfig()
	queue := Queue{
		Items: []QueueItem{
			{ID: "review-after-plan", TaskKey: "review", Title: "Chef review gate"},
		},
	}

	updated, changed, err := normalizeAndValidateQueue(queue, nil, cfg)
	if err != nil {
		t.Fatalf("normalizeAndValidateQueue error: %v", err)
	}
	if !changed {
		t.Fatal("expected synthetic review normalization to fill skill")
	}
	if got := updated.Items[0].Skill; got != "review" {
		t.Fatalf("skill = %q, want review", got)
	}
}

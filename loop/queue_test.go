package loop

import (
	"testing"

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

	queue := Queue{
		Items: []QueueItem{
			{
				ID:       "task-1",
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
	if got := updated.Items[0].Provider; got != "codex" {
		t.Fatalf("task-1 provider = %q, want codex", got)
	}
	if got := updated.Items[0].Model; got != "gpt-5.3-codex" {
		t.Fatalf("task-1 model = %q, want gpt-5.3-codex", got)
	}
	if got := updated.Items[1].Provider; got != "claude" {
		t.Fatalf("task-2 provider = %q, want claude", got)
	}
	if got := updated.Items[1].Model; got != "gpt-5.3-codex" {
		t.Fatalf("task-2 model = %q, want gpt-5.3-codex", got)
	}
}

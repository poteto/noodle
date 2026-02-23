package loop

import (
	"testing"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/taskreg"
	"github.com/poteto/noodle/skill"
)

func testQueueRegistry() taskreg.Registry {
	return taskreg.NewFromSkills([]skill.SkillMeta{
		{
			Name: "execute",
			Path: "/skills/execute",
			Frontmatter: skill.Frontmatter{
				Noodle: &skill.NoodleMeta{Schedule: "When a planned item is ready"},
			},
		},
		{
			Name: "prioritize",
			Path: "/skills/prioritize",
			Frontmatter: skill.Frontmatter{
				Noodle: &skill.NoodleMeta{Blocking: true, Schedule: "When queue is empty"},
			},
		},
		{
			Name: "reflect",
			Path: "/skills/reflect",
			Frontmatter: skill.Frontmatter{
				Noodle: &skill.NoodleMeta{Schedule: "After cook completes"},
			},
		},
	})
}

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

	updated, changed := applyQueueRoutingDefaults(queue, testQueueRegistry(), cfg)
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
	cfg.Routing.Tags["prioritize"] = config.ModelPolicy{
		Provider: "claude",
		Model:    "claude-opus-4-6",
	}

	queue := Queue{
		Items: []QueueItem{
			{
				ID:       "task-1",
				TaskKey:  "prioritize",
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

	updated, changed := applyQueueRoutingDefaults(queue, testQueueRegistry(), cfg)
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
	reg := testQueueRegistry()
	queue := Queue{
		Items: []QueueItem{
			{ID: "42", Title: "Implement fix"},
		},
	}
	backlog := []adapter.BacklogItem{
		{ID: "42", Status: adapter.BacklogStatusOpen},
	}

	updated, changed, err := normalizeAndValidateQueue(queue, backlog, reg, cfg)
	if err != nil {
		t.Fatalf("normalizeAndValidateQueue error: %v", err)
	}
	if !changed {
		t.Fatal("expected queue normalization change")
	}
	if got := updated.Items[0].TaskKey; got != "execute" {
		t.Fatalf("task_key = %q, want execute", got)
	}
	if got := updated.Items[0].Skill; got != "execute" {
		t.Fatalf("skill = %q, want execute", got)
	}
}

func TestNormalizeAndValidateQueueRejectsUnknownItem(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := testQueueRegistry()
	queue := Queue{
		Items: []QueueItem{
			{ID: "synth-1", TaskKey: "new-type", Title: "brand new type"},
		},
	}

	_, _, err := normalizeAndValidateQueue(queue, nil, reg, cfg)
	if err == nil {
		t.Fatal("expected validation error for unknown queue item")
	}
}

func TestNormalizeAndValidateQueueAllowsExecuteWhenBacklogUnavailable(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := testQueueRegistry()
	queue := Queue{
		Items: []QueueItem{
			{ID: "42", TaskKey: "execute", Title: "execute something"},
		},
	}

	_, _, err := normalizeAndValidateQueue(queue, nil, reg, cfg)
	if err != nil {
		t.Fatalf("normalizeAndValidateQueue error: %v", err)
	}
}

func TestNormalizeAndValidateQueueRejectsExecuteOutsideBacklog(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := testQueueRegistry()
	queue := Queue{
		Items: []QueueItem{
			{ID: "not-in-backlog", TaskKey: "execute", Title: "execute something"},
		},
	}
	backlog := []adapter.BacklogItem{
		{ID: "42", Status: adapter.BacklogStatusOpen},
	}

	_, _, err := normalizeAndValidateQueue(queue, backlog, reg, cfg)
	if err == nil {
		t.Fatal("expected validation error for execute item not in backlog")
	}
}

func TestNormalizeAndValidateQueueAllowsReflectTask(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := testQueueRegistry()
	queue := Queue{
		Items: []QueueItem{
			{ID: "reflect-after-cook", TaskKey: "reflect", Title: "Post-cook reflection"},
		},
	}

	updated, changed, err := normalizeAndValidateQueue(queue, nil, reg, cfg)
	if err != nil {
		t.Fatalf("normalizeAndValidateQueue error: %v", err)
	}
	if !changed {
		t.Fatal("expected normalization to fill skill")
	}
	if got := updated.Items[0].Skill; got != "reflect" {
		t.Fatalf("skill = %q, want reflect", got)
	}
}

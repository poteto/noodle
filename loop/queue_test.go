package loop

import (
	"testing"

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

	updated, changed, err := normalizeAndValidateQueue(queue, []int{42}, reg, cfg)
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
			{ID: "7", TaskKey: "execute", Title: "execute something"},
		},
	}

	_, _, err := normalizeAndValidateQueue(queue, []int{42}, reg, cfg)
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

func TestQueueXRoundTripPreservesRuntimeAndPlan(t *testing.T) {
	queue := Queue{
		Items: []QueueItem{
			{
				ID:       "task-1",
				TaskKey:  "execute",
				Provider: "claude",
				Model:    "claude-sonnet-4-6",
				Runtime:  "sprites",
				Skill:    "execute",
				Plan:     []string{"plans/27-remote-dispatchers/phase-02"},
			},
		},
	}

	roundTrip := fromQueueX(toQueueX(queue))
	item := roundTrip.Items[0]
	if item.Runtime != "sprites" {
		t.Fatalf("runtime = %q, want sprites", item.Runtime)
	}
	if len(item.Plan) != 1 || item.Plan[0] != "plans/27-remote-dispatchers/phase-02" {
		t.Fatalf("plan = %+v", item.Plan)
	}
}

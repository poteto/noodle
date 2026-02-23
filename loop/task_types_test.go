package loop

import (
	"testing"

	"github.com/poteto/noodle/config"
)

func TestConfiguredTaskTypesApplyConfigOverrides(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Adapters["plans"] = config.AdapterConfig{Skill: "plan-custom"}
	cfg.Adapters["backlog"] = config.AdapterConfig{Skill: "execute-custom"}
	cfg.Prioritize.Skill = "priority-chef"
	cfg.Phases["oops"] = "oops-custom"
	cfg.Phases["debugging"] = "repair-custom"

	plan, ok := configuredTaskTypeByKey(cfg, taskKeyPlan)
	if !ok {
		t.Fatal("expected plan task type")
	}
	if plan.Skill != "plan-custom" {
		t.Fatalf("plan skill = %q, want plan-custom", plan.Skill)
	}

	execute, ok := configuredTaskTypeByKey(cfg, taskKeyExecute)
	if !ok {
		t.Fatal("expected execute task type")
	}
	if execute.Skill != "execute-custom" {
		t.Fatalf("execute skill = %q, want execute-custom", execute.Skill)
	}

	prioritize, ok := configuredTaskTypeByKey(cfg, taskKeyPrioritize)
	if !ok {
		t.Fatal("expected prioritize task type")
	}
	if prioritize.Skill != "priority-chef" {
		t.Fatalf("prioritize skill = %q, want priority-chef", prioritize.Skill)
	}

	oops, ok := configuredTaskTypeByKey(cfg, taskKeyOops)
	if !ok {
		t.Fatal("expected oops task type")
	}
	if oops.Skill != "oops-custom" {
		t.Fatalf("oops skill = %q, want oops-custom", oops.Skill)
	}

	repair, ok := configuredTaskTypeByKey(cfg, taskKeyRepair)
	if !ok {
		t.Fatal("expected repair task type")
	}
	if repair.Skill != "repair-custom" {
		t.Fatalf("repair skill = %q, want repair-custom", repair.Skill)
	}
}

func TestTaskTypeRegistryIncludesKeySyntheticAliases(t *testing.T) {
	cfg := config.DefaultConfig()
	review, ok := configuredTaskTypeByKey(cfg, taskKeyReview)
	if !ok {
		t.Fatal("expected review task type")
	}
	if !review.Blocking {
		t.Fatal("expected review to be blocking")
	}
	if !review.Synthetic {
		t.Fatal("expected review to be synthetic")
	}
	if len(review.Aliases) == 0 {
		t.Fatal("expected review aliases")
	}
	if review.Key == "" {
		t.Fatal("expected stable key")
	}
}

func TestTaskTypeForQueueItemUsesIDPrefix(t *testing.T) {
	cfg := config.DefaultConfig()
	item := QueueItem{
		ID: "review-gate-1",
	}
	taskType, ok := taskTypeForQueueItem(cfg, item)
	if !ok {
		t.Fatal("expected id-prefix task type resolution")
	}
	if taskType.Key != taskKeyReview {
		t.Fatalf("task key = %q, want %q", taskType.Key, taskKeyReview)
	}
}

func TestTaskTypeForQueueItemDoesNotUseTitleAlias(t *testing.T) {
	cfg := config.DefaultConfig()
	item := QueueItem{
		ID:    "42",
		Title: "Chef review approval before execute",
	}
	if _, ok := taskTypeForQueueItem(cfg, item); ok {
		t.Fatal("title-only alias text should not resolve synthetic task types")
	}
}

func TestTaskTypeForQueueItemUsesExplicitTaskKey(t *testing.T) {
	cfg := config.DefaultConfig()
	item := QueueItem{
		ID:      "x-1",
		TaskKey: taskKeyMeditate,
	}
	taskType, ok := taskTypeForQueueItem(cfg, item)
	if !ok {
		t.Fatal("expected explicit task key resolution")
	}
	if taskType.Key != taskKeyMeditate {
		t.Fatalf("task key = %q, want %q", taskType.Key, taskKeyMeditate)
	}
}

package taskreg

import (
	"testing"

	"github.com/poteto/noodle/config"
)

func TestRegistryByKeyAndPrioritizeTarget(t *testing.T) {
	reg := New(config.DefaultConfig())
	taskType, ok := reg.ByKey(TaskKeyPrioritize)
	if !ok {
		t.Fatal("expected prioritize task type")
	}
	if taskType.Key != TaskKeyPrioritize {
		t.Fatalf("key = %q", taskType.Key)
	}
	if got := reg.PrioritizeTarget(); got != TaskKeyPrioritize {
		t.Fatalf("prioritize target = %q", got)
	}
}

func TestResolveQueueItem(t *testing.T) {
	reg := New(config.DefaultConfig())

	byKey, ok := reg.ResolveQueueItem(QueueItemInput{TaskKey: TaskKeyReview, ID: "x"})
	if !ok || byKey.Key != TaskKeyReview {
		t.Fatalf("resolve by key = %+v, %v", byKey, ok)
	}

	bySkill, ok := reg.ResolveQueueItem(QueueItemInput{Skill: "prioritize", ID: "x"})
	if !ok || bySkill.Key != TaskKeyPrioritize {
		t.Fatalf("resolve by skill = %+v, %v", bySkill, ok)
	}

	byAlias, ok := reg.ResolveQueueItem(QueueItemInput{ID: "42", Title: "runtime repair for monitor"})
	if !ok || byAlias.Key != TaskKeyRepair {
		t.Fatalf("resolve by alias = %+v, %v", byAlias, ok)
	}
}

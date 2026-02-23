package taskreg

import (
	"testing"

	"github.com/poteto/noodle/config"
)

func TestRegistryByKeyAndSousChefTarget(t *testing.T) {
	reg := New(config.DefaultConfig())
	taskType, ok := reg.ByKey(TaskKeySousChef)
	if !ok {
		t.Fatal("expected sous-chef task type")
	}
	if taskType.Key != TaskKeySousChef {
		t.Fatalf("key = %q", taskType.Key)
	}
	if got := reg.SousChefTarget(); got != TaskKeySousChef {
		t.Fatalf("sous-chef target = %q", got)
	}
}

func TestResolveQueueItem(t *testing.T) {
	reg := New(config.DefaultConfig())

	byKey, ok := reg.ResolveQueueItem(QueueItemInput{TaskKey: TaskKeyReview, ID: "x"})
	if !ok || byKey.Key != TaskKeyReview {
		t.Fatalf("resolve by key = %+v, %v", byKey, ok)
	}

	bySkill, ok := reg.ResolveQueueItem(QueueItemInput{Skill: "sous-chef", ID: "x"})
	if !ok || bySkill.Key != TaskKeySousChef {
		t.Fatalf("resolve by skill = %+v, %v", bySkill, ok)
	}

	byAlias, ok := reg.ResolveQueueItem(QueueItemInput{ID: "42", Title: "runtime repair for monitor"})
	if !ok || byAlias.Key != TaskKeyRepair {
		t.Fatalf("resolve by alias = %+v, %v", byAlias, ok)
	}
}

package loop

import (
	"testing"

	"github.com/poteto/noodle/internal/taskreg"
	"github.com/poteto/noodle/skill"
)

func testLoopRegistry() taskreg.Registry {
	return taskreg.NewFromSkills([]skill.SkillMeta{
		{
			Name: "prioritize",
			Path: "/skills/prioritize",
			Frontmatter: skill.Frontmatter{
				Noodle: &skill.NoodleMeta{Blocking: true, Schedule: "When queue is empty"},
			},
		},
		{
			Name: "execute",
			Path: "/skills/execute",
			Frontmatter: skill.Frontmatter{
				Noodle: &skill.NoodleMeta{Schedule: "When a planned item is ready"},
			},
		},
		{
			Name: "reflect",
			Path: "/skills/reflect",
			Frontmatter: skill.Frontmatter{
				Noodle: &skill.NoodleMeta{Schedule: "After cook completes"},
			},
		},
		{
			Name: "meditate",
			Path: "/skills/meditate",
			Frontmatter: skill.Frontmatter{
				Noodle: &skill.NoodleMeta{Schedule: "Periodically after reflects"},
			},
		},
		{
			Name: "oops",
			Path: "/skills/oops",
			Frontmatter: skill.Frontmatter{
				Noodle: &skill.NoodleMeta{Schedule: "On runtime error"},
			},
		},
	})
}

func TestIsBlockingQueueItem(t *testing.T) {
	reg := testLoopRegistry()

	if !isBlockingQueueItem(reg, QueueItem{ID: "prioritize"}) {
		t.Fatal("prioritize should be blocking")
	}
	if isBlockingQueueItem(reg, QueueItem{ID: "execute"}) {
		t.Fatal("execute should not be blocking")
	}
	if isBlockingQueueItem(reg, QueueItem{ID: "unknown-42"}) {
		t.Fatal("unknown item should not be blocking")
	}
}

func TestTaskSkillFallback(t *testing.T) {
	reg := testLoopRegistry()

	if got := taskSkill(reg, "prioritize", "fallback"); got != "prioritize" {
		t.Fatalf("got %q, want prioritize", got)
	}
	if got := taskSkill(reg, "nonexistent", "fallback"); got != "fallback" {
		t.Fatalf("got %q, want fallback", got)
	}
}

func TestResolveByExplicitTaskKey(t *testing.T) {
	reg := testLoopRegistry()
	tt, ok := reg.ResolveQueueItem(taskreg.QueueItemInput{
		ID:      "x-1",
		TaskKey: "meditate",
	})
	if !ok {
		t.Fatal("expected explicit task key resolution")
	}
	if tt.Key != "meditate" {
		t.Fatalf("task key = %q, want meditate", tt.Key)
	}
}

func TestResolveByIDPrefix(t *testing.T) {
	reg := testLoopRegistry()
	tt, ok := reg.ResolveQueueItem(taskreg.QueueItemInput{
		ID: "prioritize-20260222-123456",
	})
	if !ok {
		t.Fatal("expected id-prefix resolution")
	}
	if tt.Key != "prioritize" {
		t.Fatalf("task key = %q, want prioritize", tt.Key)
	}
}

func TestUnknownItemDoesNotResolve(t *testing.T) {
	reg := testLoopRegistry()
	if _, ok := reg.ResolveQueueItem(taskreg.QueueItemInput{
		ID:    "42",
		Title: "some ticket",
	}); ok {
		t.Fatal("unknown item should not resolve")
	}
}

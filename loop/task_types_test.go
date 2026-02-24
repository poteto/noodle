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
				Noodle: &skill.NoodleMeta{
					Permissions: skill.Permissions{Merge: boolPtr(false)},
					Schedule:    "When queue is empty",
				},
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

func TestResolveByIDPrefixNoLongerMatches(t *testing.T) {
	reg := testLoopRegistry()
	if _, ok := reg.ResolveQueueItem(taskreg.QueueItemInput{
		ID: "prioritize-20260222-123456",
	}); ok {
		t.Fatal("ID prefix fallback should not resolve — require explicit task_key")
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

func boolPtr(v bool) *bool {
	return &v
}

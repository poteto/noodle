package loop

import (
	"testing"

	"github.com/poteto/noodle/internal/taskreg"
	"github.com/poteto/noodle/skill"
)

func testLoopRegistry() taskreg.Registry {
	return taskreg.NewFromSkills([]skill.SkillMeta{
		{
			Name:        "schedule",
			Path:        "/skills/schedule",
			Frontmatter: skill.Frontmatter{Schedule: "When queue is empty"},
		},
		{
			Name:        "execute",
			Path:        "/skills/execute",
			Frontmatter: skill.Frontmatter{Schedule: "When a planned item is ready"},
		},
		{
			Name:        "reflect",
			Path:        "/skills/reflect",
			Frontmatter: skill.Frontmatter{Schedule: "After cook completes"},
		},
		{
			Name:        "meditate",
			Path:        "/skills/meditate",
			Frontmatter: skill.Frontmatter{Schedule: "Periodically after reflects"},
		},
		{
			Name:        "oops",
			Path:        "/skills/oops",
			Frontmatter: skill.Frontmatter{Schedule: "On runtime error"},
		},
		{
			Name:        "review",
			Path:        "/skills/review",
			Frontmatter: skill.Frontmatter{Schedule: "When review is needed"},
		},
	})
}

func TestResolveByExplicitTaskKey(t *testing.T) {
	reg := testLoopRegistry()
	tt, ok := reg.ResolveStage(taskreg.StageInput{
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
	if _, ok := reg.ResolveStage(taskreg.StageInput{
		ID: "schedule-20260222-123456",
	}); ok {
		t.Fatal("ID prefix fallback should not resolve — require explicit task_key")
	}
}

func TestUnknownItemDoesNotResolve(t *testing.T) {
	reg := testLoopRegistry()
	if _, ok := reg.ResolveStage(taskreg.StageInput{
		ID:    "42",
		Title: "some ticket",
	}); ok {
		t.Fatal("unknown item should not resolve")
	}
}

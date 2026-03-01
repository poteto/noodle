package taskreg

import (
	"testing"

	"github.com/poteto/noodle/skill"
)

func testSkills() []skill.SkillMeta {
	return []skill.SkillMeta{
		{
			Name: "schedule",
			Path: "/skills/schedule",
			Frontmatter: skill.Frontmatter{
				Schedule: "When the queue is empty",
			},
		},
		{
			Name: "execute",
			Path: "/skills/execute",
			Frontmatter: skill.Frontmatter{
				Schedule: "When a planned item is ready",
			},
		},
		{
			Name: "reflect",
			Path: "/skills/reflect",
			Frontmatter: skill.Frontmatter{
				Schedule: "After a cook session completes",
			},
		},
		{
			Name: "debugging",
			Path: "/skills/debugging",
			Frontmatter: skill.Frontmatter{
				Name: "debugging",
			},
		},
	}
}

func TestNewFromSkillsExcludesUtilitySkills(t *testing.T) {
	reg := NewFromSkills(testSkills())
	all := reg.All()
	if len(all) != 3 {
		t.Fatalf("types = %d, want 3", len(all))
	}
	if _, ok := reg.ByKey("debugging"); ok {
		t.Fatal("utility skill should not be in registry")
	}
}

func TestByKey(t *testing.T) {
	reg := NewFromSkills(testSkills())
	tt, ok := reg.ByKey("schedule")
	if !ok {
		t.Fatal("expected schedule")
	}
	if tt.Key != "schedule" {
		t.Fatalf("key = %q", tt.Key)
	}
}

func TestResolveStageByTaskKey(t *testing.T) {
	reg := NewFromSkills(testSkills())
	tt, ok := reg.ResolveStage(StageInput{TaskKey: "execute", ID: "x"})
	if !ok || tt.Key != "execute" {
		t.Fatalf("resolve by task_key = %+v, %v", tt, ok)
	}
}

func TestResolveStageBySkill(t *testing.T) {
	reg := NewFromSkills(testSkills())
	tt, ok := reg.ResolveStage(StageInput{Skill: "schedule", ID: "x"})
	if !ok || tt.Key != "schedule" {
		t.Fatalf("resolve by skill = %+v, %v", tt, ok)
	}
}

func TestResolveStageByIDPrefixNoLongerMatches(t *testing.T) {
	reg := NewFromSkills(testSkills())
	if _, ok := reg.ResolveStage(StageInput{ID: "schedule-20260222-123456-1"}); ok {
		t.Fatal("ID prefix fallback should not resolve — require explicit task_key")
	}
}

func TestResolveStageUnknown(t *testing.T) {
	reg := NewFromSkills(testSkills())
	if _, ok := reg.ResolveStage(StageInput{ID: "42", Title: "some ticket"}); ok {
		t.Fatal("unknown item should not resolve")
	}
}

func TestResolveStageByExactID(t *testing.T) {
	reg := NewFromSkills(testSkills())
	tt, ok := reg.ResolveStage(StageInput{ID: "reflect"})
	if !ok || tt.Key != "reflect" {
		t.Fatalf("resolve by exact id = %+v, %v", tt, ok)
	}
}

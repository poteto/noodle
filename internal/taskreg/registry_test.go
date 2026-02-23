package taskreg

import (
	"testing"

	"github.com/poteto/noodle/skill"
)

func testSkills() []skill.SkillMeta {
	return []skill.SkillMeta{
		{
			Name: "prioritize",
			Path: "/skills/prioritize",
			Frontmatter: skill.Frontmatter{
				Noodle: &skill.NoodleMeta{
					Blocking: true,
					Schedule: "When the queue is empty",
				},
			},
		},
		{
			Name: "execute",
			Path: "/skills/execute",
			Frontmatter: skill.Frontmatter{
				Noodle: &skill.NoodleMeta{
					Schedule: "When a planned item is ready",
				},
			},
		},
		{
			Name: "reflect",
			Path: "/skills/reflect",
			Frontmatter: skill.Frontmatter{
				Noodle: &skill.NoodleMeta{
					Schedule: "After a cook session completes",
				},
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
	tt, ok := reg.ByKey("prioritize")
	if !ok {
		t.Fatal("expected prioritize")
	}
	if tt.Key != "prioritize" {
		t.Fatalf("key = %q", tt.Key)
	}
	if !tt.Blocking {
		t.Fatal("expected blocking")
	}
}

func TestResolveQueueItemByTaskKey(t *testing.T) {
	reg := NewFromSkills(testSkills())
	tt, ok := reg.ResolveQueueItem(QueueItemInput{TaskKey: "execute", ID: "x"})
	if !ok || tt.Key != "execute" {
		t.Fatalf("resolve by task_key = %+v, %v", tt, ok)
	}
}

func TestResolveQueueItemBySkill(t *testing.T) {
	reg := NewFromSkills(testSkills())
	tt, ok := reg.ResolveQueueItem(QueueItemInput{Skill: "prioritize", ID: "x"})
	if !ok || tt.Key != "prioritize" {
		t.Fatalf("resolve by skill = %+v, %v", tt, ok)
	}
}

func TestResolveQueueItemByIDPrefixNoLongerMatches(t *testing.T) {
	reg := NewFromSkills(testSkills())
	if _, ok := reg.ResolveQueueItem(QueueItemInput{ID: "prioritize-20260222-123456-1"}); ok {
		t.Fatal("ID prefix fallback should not resolve — require explicit task_key")
	}
}

func TestResolveQueueItemUnknown(t *testing.T) {
	reg := NewFromSkills(testSkills())
	if _, ok := reg.ResolveQueueItem(QueueItemInput{ID: "42", Title: "some ticket"}); ok {
		t.Fatal("unknown item should not resolve")
	}
}

func TestResolveQueueItemByExactID(t *testing.T) {
	reg := NewFromSkills(testSkills())
	tt, ok := reg.ResolveQueueItem(QueueItemInput{ID: "reflect"})
	if !ok || tt.Key != "reflect" {
		t.Fatalf("resolve by exact id = %+v, %v", tt, ok)
	}
}

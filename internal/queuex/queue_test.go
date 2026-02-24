package queuex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/taskreg"
	"github.com/poteto/noodle/skill"
)

func testRegistry() taskreg.Registry {
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
	})
}

func TestReadMissingAndEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "queue.json")

	q, err := Read(path)
	if err != nil {
		t.Fatalf("read missing: %v", err)
	}
	if len(q.Items) != 0 {
		t.Fatalf("items = %d", len(q.Items))
	}

	if err := os.WriteFile(path, []byte("\n  \n"), 0o644); err != nil {
		t.Fatalf("write empty: %v", err)
	}
	q, err = Read(path)
	if err != nil {
		t.Fatalf("read empty: %v", err)
	}
	if len(q.Items) != 0 {
		t.Fatalf("items = %d", len(q.Items))
	}
}

func TestReadSupportsWrappedAndLegacyArray(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "queue.json")

	wrapped := `{"items":[{"id":"1","task_key":"execute"}]}`
	if err := os.WriteFile(path, []byte(wrapped), 0o644); err != nil {
		t.Fatalf("write wrapped: %v", err)
	}
	q, err := Read(path)
	if err != nil {
		t.Fatalf("read wrapped: %v", err)
	}
	if len(q.Items) != 1 || q.Items[0].ID != "1" {
		t.Fatalf("wrapped items = %+v", q.Items)
	}

	legacy := `[{"id":"2","task_key":"execute"}]`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write legacy: %v", err)
	}
	q, err = Read(path)
	if err != nil {
		t.Fatalf("read legacy: %v", err)
	}
	if len(q.Items) != 1 || q.Items[0].ID != "2" {
		t.Fatalf("legacy items = %+v", q.Items)
	}
}

func TestReadStrictRejectsLegacyArray(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "queue.json")
	legacy := `[{\"id\":\"2\",\"task_key\":\"execute\"}]`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write legacy: %v", err)
	}
	if _, err := ReadStrict(path); err == nil {
		t.Fatal("expected strict read to reject legacy array")
	}
}

func TestNormalizeAndValidateRejectsDuplicateIDs(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := testRegistry()
	queue := Queue{Items: []Item{{ID: "1", TaskKey: "execute"}, {ID: "1", TaskKey: "execute"}}}

	_, _, err := NormalizeAndValidate(queue, nil, reg, cfg)
	if err == nil || !strings.Contains(err.Error(), "appears more than once") {
		t.Fatalf("expected duplicate id error, got %v", err)
	}
}

func TestNormalizeAndValidateAppliesTaskDefaults(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := testRegistry()
	queue := Queue{Items: []Item{{ID: "1", Title: "implement feature"}}}

	got, changed, err := NormalizeAndValidate(queue, []int{1}, reg, cfg)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if !changed {
		t.Fatal("expected changed")
	}
	if got.Items[0].TaskKey != "execute" {
		t.Fatalf("task key = %q", got.Items[0].TaskKey)
	}
	if strings.TrimSpace(got.Items[0].Skill) == "" {
		t.Fatalf("skill not populated: %+v", got.Items[0])
	}
}

func TestReadParsesRuntimeField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "queue.json")

	wrapped := `{"items":[{"id":"1","task_key":"execute","runtime":"sprites"}]}`
	if err := os.WriteFile(path, []byte(wrapped), 0o644); err != nil {
		t.Fatalf("write wrapped: %v", err)
	}
	q, err := Read(path)
	if err != nil {
		t.Fatalf("read queue: %v", err)
	}
	if got := q.Items[0].Runtime; got != "sprites" {
		t.Fatalf("runtime = %q, want sprites", got)
	}
}

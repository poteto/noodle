package skill

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveFirstMatchWins(t *testing.T) {
	project := t.TempDir()
	user := t.TempDir()

	mustMkdirAll(t, filepath.Join(project, "review"))
	mustMkdirAll(t, filepath.Join(user, "review"))
	mustWriteFile(t, filepath.Join(project, "review", "SKILL.md"), "# review project")
	mustWriteFile(t, filepath.Join(user, "review", "SKILL.md"), "# review user")

	resolver := Resolver{
		SearchPaths: []string{project, user},
	}

	resolved, err := resolver.Resolve("review")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.SourcePath != project {
		t.Fatalf("source path = %q, want %q", resolved.SourcePath, project)
	}
	if !strings.HasSuffix(resolved.Path, filepath.Join("review")) {
		t.Fatalf("resolved path = %q", resolved.Path)
	}
}

func TestResolveMissingSkillReturnsError(t *testing.T) {
	resolver := Resolver{
		SearchPaths: []string{t.TempDir()},
	}

	_, err := resolver.Resolve("does-not-exist")
	if err == nil {
		t.Fatal("expected missing skill error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveMissingSkillReturnsErrNotFound(t *testing.T) {
	resolver := Resolver{
		SearchPaths: []string{t.TempDir()},
	}

	_, err := resolver.Resolve("nonexistent-skill")
	if err == nil {
		t.Fatal("expected error for missing skill")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
	if !strings.Contains(err.Error(), "nonexistent-skill") {
		t.Fatalf("error should contain skill name, got: %v", err)
	}
}

func TestResolveStatErrorIsNotErrNotFound(t *testing.T) {
	// Make the search path directory itself non-readable so that os.Stat on
	// candidates returns a permission error (not ErrNotExist).
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "broken-skill")
	mustMkdirAll(t, skillDir)
	mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), "# Broken")

	// Remove read+execute permission from the search path so stat fails.
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	resolver := Resolver{SearchPaths: []string{dir}}
	_, err := resolver.Resolve("broken-skill")
	if err == nil {
		t.Fatal("expected error for permission-denied search path")
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatalf("permission error should not be ErrNotFound: %v", err)
	}
}

func TestResolveHandlesEmptySearchPaths(t *testing.T) {
	resolver := Resolver{
		SearchPaths: []string{"", "   "},
	}

	_, err := resolver.Resolve("review")
	if err == nil {
		t.Fatal("expected missing skill error")
	}
}

func TestResolveSkipsMissingSearchPathDirectories(t *testing.T) {
	project := t.TempDir()
	missing := filepath.Join(t.TempDir(), "missing-dir")
	mustMkdirAll(t, filepath.Join(project, "review"))
	mustWriteFile(t, filepath.Join(project, "review", "SKILL.md"), "# review")

	resolver := Resolver{
		SearchPaths: []string{missing, project},
	}

	resolved, err := resolver.Resolve("review")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.SourcePath != project {
		t.Fatalf("source path = %q, want %q", resolved.SourcePath, project)
	}
}

func TestListReturnsResolvedSkillsWithPrecedence(t *testing.T) {
	project := t.TempDir()
	user := t.TempDir()

	mustMkdirAll(t, filepath.Join(project, "review"))
	mustWriteFile(t, filepath.Join(project, "review", "SKILL.md"), "# review")
	mustMkdirAll(t, filepath.Join(user, "review"))
	mustWriteFile(t, filepath.Join(user, "review", "SKILL.md"), "# review user")
	mustMkdirAll(t, filepath.Join(user, "debug"))
	mustWriteFile(t, filepath.Join(user, "debug", "SKILL.md"), "# debug")

	resolver := Resolver{
		SearchPaths: []string{project, user},
	}

	skills, err := resolver.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("skills len = %d, want 2", len(skills))
	}

	if skills[0].Name != "review" {
		t.Fatalf("skills[0].name = %q, want review", skills[0].Name)
	}
	if skills[0].SourcePath != project {
		t.Fatalf("skills[0].source = %q, want %q", skills[0].SourcePath, project)
	}
	if !skills[0].HasSkillMD {
		t.Fatal("skills[0] should have SKILL.md")
	}

	if skills[1].Name != "debug" {
		t.Fatalf("skills[1].name = %q, want debug", skills[1].Name)
	}
	if skills[1].SourcePath != user {
		t.Fatalf("skills[1].source = %q, want %q", skills[1].SourcePath, user)
	}
	if !skills[1].HasSkillMD {
		t.Fatal("skills[1] should have SKILL.md")
	}
}

func TestResolveIgnoresDirectoryWithoutSkillFile(t *testing.T) {
	project := t.TempDir()
	user := t.TempDir()

	mustMkdirAll(t, filepath.Join(project, "review"))
	mustMkdirAll(t, filepath.Join(user, "review"))
	mustWriteFile(t, filepath.Join(user, "review", "SKILL.md"), "# review user")

	resolver := Resolver{
		SearchPaths: []string{project, user},
	}

	resolved, err := resolver.Resolve("review")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.SourcePath != user {
		t.Fatalf("source path = %q, want %q", resolved.SourcePath, user)
	}
}

func TestListIgnoresDirectoryWithoutSkillFile(t *testing.T) {
	project := t.TempDir()
	user := t.TempDir()

	mustMkdirAll(t, filepath.Join(project, "review"))
	mustMkdirAll(t, filepath.Join(user, "review"))
	mustWriteFile(t, filepath.Join(user, "review", "SKILL.md"), "# review user")

	resolver := Resolver{
		SearchPaths: []string{project, user},
	}

	skills, err := resolver.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("skills len = %d, want 1", len(skills))
	}
	if skills[0].Name != "review" {
		t.Fatalf("skills[0].name = %q, want review", skills[0].Name)
	}
	if skills[0].SourcePath != user {
		t.Fatalf("skills[0].source = %q, want %q", skills[0].SourcePath, user)
	}
}

func TestDiscoverTaskTypesFiltersCorrectly(t *testing.T) {
	dir := t.TempDir()

	// Task type skill (has noodle: block)
	mustMkdirAll(t, filepath.Join(dir, "schedule"))
	mustWriteFile(t, filepath.Join(dir, "schedule", "SKILL.md"), `---
name: schedule
description: Queue scheduler
noodle:
  permissions:
    merge: false
  schedule: "When the queue is empty"
---
# Schedule
`)

	// Utility skill (no noodle: block)
	mustMkdirAll(t, filepath.Join(dir, "debugging"))
	mustWriteFile(t, filepath.Join(dir, "debugging", "SKILL.md"), `---
name: debugging
description: Systematic debugging
---
# Debugging
`)

	// Another task type
	mustMkdirAll(t, filepath.Join(dir, "execute"))
	mustWriteFile(t, filepath.Join(dir, "execute", "SKILL.md"), `---
name: execute
description: Implementation
noodle:
  schedule: "When a planned item is ready"
---
# Execute
`)

	resolver := Resolver{SearchPaths: []string{dir}}

	taskTypes, err := resolver.DiscoverTaskTypes()
	if err != nil {
		t.Fatal(err)
	}
	if len(taskTypes) != 2 {
		t.Fatalf("task types = %d, want 2", len(taskTypes))
	}

	names := map[string]bool{}
	for _, tt := range taskTypes {
		names[tt.Name] = true
		if !tt.Frontmatter.IsTaskType() {
			t.Fatalf("%s should be a task type", tt.Name)
		}
	}
	if !names["schedule"] || !names["execute"] {
		t.Fatalf("expected schedule and execute, got %v", names)
	}
}

func TestResolveWithMetaParsesNoodleFrontmatter(t *testing.T) {
	dir := t.TempDir()
	mustMkdirAll(t, filepath.Join(dir, "deploy"))
	mustWriteFile(t, filepath.Join(dir, "deploy", "SKILL.md"), `---
name: deploy
description: Deploy to production
noodle:
  permissions:
    merge: true
  schedule: "After successful execute"
---
# Deploy
`)

	resolver := Resolver{SearchPaths: []string{dir}}
	meta, err := resolver.ResolveWithMeta("deploy")
	if err != nil {
		t.Fatal(err)
	}
	if meta.Name != "deploy" {
		t.Fatalf("name = %q", meta.Name)
	}
	if !meta.Frontmatter.IsTaskType() {
		t.Fatal("expected task type")
	}
	if meta.Frontmatter.Noodle.Schedule != "After successful execute" {
		t.Fatalf("schedule = %q", meta.Frontmatter.Noodle.Schedule)
	}
}

func TestListWithMetaIncludesAllSkills(t *testing.T) {
	dir := t.TempDir()

	mustMkdirAll(t, filepath.Join(dir, "task-skill"))
	mustWriteFile(t, filepath.Join(dir, "task-skill", "SKILL.md"), `---
name: task-skill
noodle:
  schedule: "Always"
---
# Task
`)

	mustMkdirAll(t, filepath.Join(dir, "util-skill"))
	mustWriteFile(t, filepath.Join(dir, "util-skill", "SKILL.md"), `---
name: util-skill
description: Utility
---
# Util
`)

	resolver := Resolver{SearchPaths: []string{dir}}
	all, err := resolver.ListWithMeta()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("all = %d, want 2", len(all))
	}

	taskCount := 0
	for _, m := range all {
		if m.Frontmatter.IsTaskType() {
			taskCount++
		}
	}
	if taskCount != 1 {
		t.Fatalf("task types = %d, want 1", taskCount)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

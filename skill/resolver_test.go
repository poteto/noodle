package skill

import (
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

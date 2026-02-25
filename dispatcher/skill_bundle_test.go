package dispatcher

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poteto/noodle/skill"
)

func TestLoadSkillBundleCodexTruncatesReferences(t *testing.T) {
	searchPath := t.TempDir()
	skillDir := filepath.Join(searchPath, "demo-skill")
	if err := os.MkdirAll(filepath.Join(skillDir, "references"), 0o755); err != nil {
		t.Fatalf("mkdir references: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("Core instructions"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "references", "small.md"), []byte("small"), 0o644); err != nil {
		t.Fatalf("write small ref: %v", err)
	}
	large := strings.Repeat("x", codexSkillRefsLimitBytes+32)
	if err := os.WriteFile(filepath.Join(skillDir, "references", "large.md"), []byte(large), 0o644); err != nil {
		t.Fatalf("write large ref: %v", err)
	}

	resolver := skill.Resolver{SearchPaths: []string{searchPath}}
	loaded, err := loadSkillBundle(resolver, "codex", "demo-skill")
	if err != nil {
		t.Fatalf("load skill bundle: %v", err)
	}
	if len(loaded.Warnings) == 0 {
		t.Fatal("expected truncation warning")
	}
	if !strings.Contains(loaded.SystemPrompt, "small.md") {
		t.Fatalf("expected included small reference, got:\n%s", loaded.SystemPrompt)
	}
	if strings.Contains(loaded.SystemPrompt, large[:256]) {
		t.Fatal("expected large reference content to be omitted")
	}
	if !strings.Contains(loaded.SystemPrompt, "Omitted files due 50KB") {
		t.Fatalf("expected omission note, got:\n%s", loaded.SystemPrompt)
	}
}

func TestLoadSkillBundleClaudeIncludesReferences(t *testing.T) {
	searchPath := t.TempDir()
	skillDir := filepath.Join(searchPath, "demo-skill")
	if err := os.MkdirAll(filepath.Join(skillDir, "references"), 0o755); err != nil {
		t.Fatalf("mkdir references: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("Core instructions"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "references", "guide.md"), []byte("guide text"), 0o644); err != nil {
		t.Fatalf("write ref: %v", err)
	}

	resolver := skill.Resolver{SearchPaths: []string{searchPath}}
	loaded, err := loadSkillBundle(resolver, "claude", "demo-skill")
	if err != nil {
		t.Fatalf("load skill bundle: %v", err)
	}
	if len(loaded.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", loaded.Warnings)
	}
	if !strings.Contains(loaded.SystemPrompt, "## SKILL.md") {
		t.Fatalf("missing SKILL.md section:\n%s", loaded.SystemPrompt)
	}
	if !strings.Contains(loaded.SystemPrompt, "## references/guide.md") {
		t.Fatalf("missing references section:\n%s", loaded.SystemPrompt)
	}
}

func TestLoadExecuteBundleCombinesSkills(t *testing.T) {
	searchPath := t.TempDir()
	for _, name := range []string{"execute", "todo"} {
		dir := filepath.Join(searchPath, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(name+" instructions"), 0o644); err != nil {
			t.Fatalf("write %s SKILL.md: %v", name, err)
		}
	}

	resolver := skill.Resolver{SearchPaths: []string{searchPath}}
	loaded, err := loadExecuteBundle(resolver, "claude", "execute", "todo")
	if err != nil {
		t.Fatalf("load execute bundle: %v", err)
	}
	if !strings.Contains(loaded.SystemPrompt, "execute instructions") {
		t.Fatal("missing methodology skill content")
	}
	if !strings.Contains(loaded.SystemPrompt, "todo instructions") {
		t.Fatal("missing domain skill content")
	}
	idx1 := strings.Index(loaded.SystemPrompt, "execute instructions")
	idx2 := strings.Index(loaded.SystemPrompt, "todo instructions")
	if idx1 >= idx2 {
		t.Fatal("methodology should appear before domain skill")
	}
}

func TestLoadExecuteBundleMissingDomainIsWarning(t *testing.T) {
	searchPath := t.TempDir()
	dir := filepath.Join(searchPath, "execute")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("execute instructions"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	resolver := skill.Resolver{SearchPaths: []string{searchPath}}
	loaded, err := loadExecuteBundle(resolver, "claude", "execute", "nonexistent")
	if err != nil {
		t.Fatalf("load execute bundle: %v", err)
	}
	if len(loaded.Warnings) == 0 {
		t.Fatal("expected warning for missing domain skill")
	}
	if !strings.Contains(loaded.Warnings[0], "nonexistent") {
		t.Fatalf("warning should mention skill name: %s", loaded.Warnings[0])
	}
	if !strings.Contains(loaded.SystemPrompt, "execute instructions") {
		t.Fatal("methodology skill should still be present")
	}
}

func TestLoadExecuteBundleSameSkillNoDuplicate(t *testing.T) {
	searchPath := t.TempDir()
	dir := filepath.Join(searchPath, "execute")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("execute instructions"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	resolver := skill.Resolver{SearchPaths: []string{searchPath}}
	loaded, err := loadExecuteBundle(resolver, "claude", "execute", "execute")
	if err != nil {
		t.Fatalf("load execute bundle: %v", err)
	}
	count := strings.Count(loaded.SystemPrompt, "execute instructions")
	if count != 1 {
		t.Fatalf("expected skill content once, found %d times", count)
	}
}

func TestResolveTemplateVars(t *testing.T) {
	tmpl := "run --session={{session}} --repo={{repo}} --prompt={{prompt}} --skill={{skill}} --brief={{brief}}"
	vars := map[string]string{
		"session": "noodle-exec-abc",
		"repo":    "/path/to/repo",
		"prompt":  "/tmp/prompt.txt",
		"skill":   "execute",
		"brief":   "/path/.noodle/mise.json",
	}
	result := resolveTemplateVars(tmpl, vars)
	expected := "run --session=noodle-exec-abc --repo=/path/to/repo --prompt=/tmp/prompt.txt --skill=execute --brief=/path/.noodle/mise.json"
	if result != expected {
		t.Fatalf("resolved = %q, want %q", result, expected)
	}
}

func TestResolveTemplateVarsNoQuoting(t *testing.T) {
	tmpl := "cmd '{{repo}}'"
	result := resolveTemplateVars(tmpl, map[string]string{"repo": "/path with spaces/repo"})
	if result != "cmd '/path with spaces/repo'" {
		t.Fatalf("should not add extra quoting: %q", result)
	}
}

func TestLoadSkillBundleStripsFrontmatter(t *testing.T) {
	searchPath := t.TempDir()
	dir := filepath.Join(searchPath, "fm-skill")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "---\nname: fm-skill\nnoodle:\n  schedule: \"always\"\n---\n\n# FM Skill\nBody here"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	resolver := skill.Resolver{SearchPaths: []string{searchPath}}
	loaded, err := loadSkillBundle(resolver, "claude", "fm-skill")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if strings.Contains(loaded.SystemPrompt, "---") {
		t.Fatal("frontmatter YAML markers should be stripped")
	}
	if !strings.Contains(loaded.SystemPrompt, "# FM Skill") {
		t.Fatal("body content should be present")
	}
}

func TestLoadSkillBundleMissingSkillIsWarning(t *testing.T) {
	resolver := skill.Resolver{SearchPaths: []string{t.TempDir()}}
	loaded, err := loadSkillBundle(resolver, "claude", "nonexistent-skill")
	if err != nil {
		t.Fatalf("missing skill should not be a hard error: %v", err)
	}
	if loaded.SystemPrompt != "" {
		t.Fatalf("expected empty system prompt, got: %s", loaded.SystemPrompt)
	}
	if len(loaded.Warnings) == 0 {
		t.Fatal("expected warning for missing skill")
	}
	if !strings.Contains(loaded.Warnings[0], "nonexistent-skill") {
		t.Fatalf("warning should mention skill name: %s", loaded.Warnings[0])
	}
}

func TestLoadSkillBundleFilesystemErrorIsHardFail(t *testing.T) {
	// Remove read+execute permission from the search path so stat on
	// candidates returns a permission error, not ErrNotFound.
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "broken-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Broken"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	resolver := skill.Resolver{SearchPaths: []string{dir}}
	_, err := loadSkillBundle(resolver, "claude", "broken-skill")
	if err == nil {
		t.Fatal("expected hard error for filesystem failure")
	}
	if errors.Is(err, skill.ErrNotFound) {
		t.Fatalf("filesystem error should not be ErrNotFound: %v", err)
	}
}

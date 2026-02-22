package spawner

import (
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

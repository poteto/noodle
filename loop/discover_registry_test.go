package loop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/poteto/noodle/config"
)

func TestDiscoverRegistryExpandsHomeSkillPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	skillDir := filepath.Join(homeDir, ".noodle", "skills", "deploy")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("create skill directory: %v", err)
	}
	content := `---
name: deploy
description: Deployment flow
noodle:
  schedule: "After execute succeeds"
---
# Deploy
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Skills.Paths = []string{"~/.noodle/skills"}

	registry, err := discoverRegistry(t.TempDir(), cfg)
	if err != nil {
		t.Fatalf("discover registry: %v", err)
	}
	if _, ok := registry.ByKey("deploy"); !ok {
		t.Fatal("expected deploy task type from ~/.noodle/skills")
	}
}

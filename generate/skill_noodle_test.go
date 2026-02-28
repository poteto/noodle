package generate

import (
	"strings"
	"testing"
)

func TestGenerateSkillContent(t *testing.T) {
	generated, err := GenerateSkillContent()
	if err != nil {
		t.Fatalf("GenerateSkillContent: %v", err)
	}
	if !strings.Contains(generated, "# Noodle") {
		t.Fatal("generated skill missing Noodle heading")
	}
	if !strings.Contains(generated, "## Config Reference") {
		t.Fatal("generated skill missing Config Reference section")
	}
}

func TestGeneratedSkillContainsAllConfigFields(t *testing.T) {
	content, err := GenerateSkillContent()
	if err != nil {
		t.Fatalf("GenerateSkillContent: %v", err)
	}

	requiredFields := []string{
		"autonomy",
		"routing.defaults.provider",
		"routing.defaults.model",
		"skills.paths",
		"concurrency.max_cooks",
	}

	for _, field := range requiredFields {
		if !strings.Contains(content, field) {
			t.Errorf("generated skill missing config field %q", field)
		}
	}
}

func TestGeneratedSkillContainsAllCommands(t *testing.T) {
	content, err := GenerateSkillContent()
	if err != nil {
		t.Fatalf("GenerateSkillContent: %v", err)
	}

	requiredCommands := []string{
		"noodle start",
		"noodle status",
		"noodle debug",
		"noodle skills",
		"noodle worktree",
		"noodle mise",
		"noodle stamp",
		"noodle dispatch",
		"noodle schema",
	}

	for _, cmd := range requiredCommands {
		if !strings.Contains(content, cmd) {
			t.Errorf("generated skill missing command %q", cmd)
		}
	}
}

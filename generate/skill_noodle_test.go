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
	if !strings.Contains(generated, "## Configuration") {
		t.Fatal("generated skill missing Configuration section")
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
		"noodle skills",
		"noodle worktree",
		"noodle schema",
		"noodle event",
		"noodle reset",
	}

	for _, cmd := range requiredCommands {
		if !strings.Contains(content, cmd) {
			t.Errorf("generated skill missing command %q", cmd)
		}
	}
}

func TestGeneratedSkillContainsSections(t *testing.T) {
	content, err := GenerateSkillContent()
	if err != nil {
		t.Fatalf("GenerateSkillContent: %v", err)
	}

	sections := []string{
		"## How the Loop Works",
		"## Skills",
		"## Configuration",
		"## CLI",
		"## Troubleshooting",
		"## References",
	}

	for _, section := range sections {
		if !strings.Contains(content, section) {
			t.Errorf("generated skill missing section %q", section)
		}
	}
}

func TestGeneratedSkillNoRemovedContracts(t *testing.T) {
	content, err := GenerateSkillContent()
	if err != nil {
		t.Fatalf("GenerateSkillContent: %v", err)
	}

	removed := []string{
		"schedule.run",
		"schedule.model",
		"PendingApproval",
		"isScheduleTarget",
		"steerFallbackRespawn",
		"promptItemRegexp",
	}

	for _, term := range removed {
		if strings.Contains(content, term) {
			t.Errorf("generated skill still references removed contract %q", term)
		}
	}
}

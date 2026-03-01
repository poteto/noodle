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
		"mode",
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
		"noodle skills",
		"noodle worktree",
		"noodle schema",
	}

	for _, cmd := range requiredCommands {
		if !strings.Contains(content, cmd) {
			t.Errorf("generated skill missing command %q", cmd)
		}
	}
}

func TestGeneratedSkillContainsV2Contracts(t *testing.T) {
	content, err := GenerateSkillContent()
	if err != nil {
		t.Fatalf("GenerateSkillContent: %v", err)
	}

	v2Sections := []string{
		"## Mode Contract",
		"## Runtime Capabilities",
		"## Canonical State Model",
		"## Control Commands",
		"## Dispatch and Projection",
	}

	for _, section := range v2Sections {
		if !strings.Contains(content, section) {
			t.Errorf("generated skill missing V2 contract section %q", section)
		}
	}

	v2Vocabulary := []string{
		"supervised",
		"mode_epoch",
		"steerable",
		"polling",
		"remote_sync",
		"requested_mode",
		"effective_mode",
		"RouteCompletion",
		"PlanDispatches",
	}

	for _, term := range v2Vocabulary {
		if !strings.Contains(content, term) {
			t.Errorf("generated skill missing V2 vocabulary term %q", term)
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

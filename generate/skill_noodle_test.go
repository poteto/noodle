package generate

import (
	"os"
	"strings"
	"testing"
)

func TestNoodleSkillSnapshot(t *testing.T) {
	generated, err := GenerateSkillContent()
	if err != nil {
		t.Fatalf("GenerateSkillContent: %v", err)
	}

	committed, err := os.ReadFile("../.agents/skills/noodle/SKILL.md")
	if err != nil {
		t.Fatalf("read committed SKILL.md: %v", err)
	}

	if string(committed) != generated {
		// Find first differing line for a useful error message
		committedLines := strings.Split(string(committed), "\n")
		generatedLines := strings.Split(generated, "\n")
		for i := 0; i < len(committedLines) || i < len(generatedLines); i++ {
			var cl, gl string
			if i < len(committedLines) {
				cl = committedLines[i]
			}
			if i < len(generatedLines) {
				gl = generatedLines[i]
			}
			if cl != gl {
				t.Fatalf("noodle skill is out of date (first diff at line %d).\n"+
					"  committed: %q\n"+
					"  generated: %q\n"+
					"Run `go generate ./generate/...` to update.",
					i+1, cl, gl,
				)
			}
		}
		t.Fatal("noodle skill is out of date. Run `go generate ./generate/...` to update.")
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
		"schedule.run",
		"concurrency.max_cooks",
		"plans.on_done",
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
		"noodle plan",
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

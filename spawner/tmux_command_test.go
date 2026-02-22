package spawner

import (
	"strings"
	"testing"
)

func TestBuildProviderCommandClaude(t *testing.T) {
	req := SpawnRequest{
		Name:           "cook-a",
		Prompt:         "Say ok",
		Provider:       "claude",
		Model:          "claude-sonnet-4-6",
		ReasoningLevel: "medium",
		MaxTurns:       5,
		BudgetCap:      2.5,
	}
	command := buildProviderCommand(req, "/tmp/prompt.txt", "claude", "skill-system")

	wants := []string{
		"'claude'",
		"'--output-format' 'stream-json'",
		"'--model' 'claude-sonnet-4-6'",
		"'--max-turns' '5'",
		"'--max-budget-usd' '2.50'",
		"'--append-system-prompt' 'skill-system'",
		"< '/tmp/prompt.txt'",
	}
	for _, want := range wants {
		if !strings.Contains(command, want) {
			t.Fatalf("command missing %q:\n%s", want, command)
		}
	}
}

func TestBuildProviderCommandCodex(t *testing.T) {
	req := SpawnRequest{
		Name:         "cook-b",
		Prompt:       "Say ok",
		Provider:     "codex",
		Model:        "gpt-5.3-codex",
		WorktreePath: ".worktrees/phase-06-spawner",
	}
	command := buildProviderCommand(req, "/tmp/prompt.txt", "codex", "")

	wants := []string{
		"'codex' 'exec'",
		"'--skip-git-repo-check'",
		"'--full-auto'",
		"'--sandbox' 'workspace-write'",
		"'--json'",
		"'--model' 'gpt-5.3-codex'",
		"< '/tmp/prompt.txt'",
	}
	for _, want := range wants {
		if !strings.Contains(command, want) {
			t.Fatalf("command missing %q:\n%s", want, command)
		}
	}
}

func TestBuildPipelineCommand(t *testing.T) {
	command := buildPipelineCommand(
		"'claude' -p < '/tmp/prompt.txt' 2>&1",
		"/usr/local/bin/noodle",
		"/tmp/stamped.ndjson",
		"/tmp/canonical.ndjson",
	)
	if !strings.Contains(command, "stamp --output '/tmp/stamped.ndjson' --events '/tmp/canonical.ndjson'") {
		t.Fatalf("unexpected pipeline command: %s", command)
	}
}

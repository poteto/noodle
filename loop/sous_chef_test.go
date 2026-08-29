package loop

import (
	"strings"
	"testing"
)

func TestBuildOrderTaskTypesPromptIncludesKeyAndSchedule(t *testing.T) {
	prompt := buildOrderTaskTypesPrompt([]TaskType{
		{
			Key:      "execute",
			Schedule: "When a backlog item is ready",
		},
	})

	if !strings.Contains(prompt, "- execute: When a backlog item is ready") {
		t.Fatalf("unexpected prompt: %q", prompt)
	}
}

func TestBuildOrderTaskTypesPromptEmpty(t *testing.T) {
	prompt := buildOrderTaskTypesPrompt(nil)
	if !strings.Contains(prompt, "Task types you may schedule:") {
		t.Fatalf("missing prompt header: %q", prompt)
	}
	if !strings.Contains(prompt, "- (none configured)") {
		t.Fatalf("missing empty marker: %q", prompt)
	}
}

// The schedule task type is transient and self-scheduling is forbidden: the
// scheduler must never see itself advertised as a task it may emit, even
// when the registry carries a "schedule" entry (it always does — the
// schedule skill itself has schedule frontmatter).
func TestBuildOrderTaskTypesPromptExcludesSelfSchedule(t *testing.T) {
	prompt := buildOrderTaskTypesPrompt([]TaskType{
		{Key: "schedule", Schedule: "When orders are empty"},
		{Key: "execute", Schedule: "When ready"},
	})

	if strings.Contains(prompt, "- schedule:") {
		t.Fatalf("prompt should not self-advertise the schedule task type: %q", prompt)
	}
	if !strings.Contains(prompt, "- execute: When ready") {
		t.Fatalf("prompt should still list non-schedule task types: %q", prompt)
	}
}

func TestBuildOrderTaskTypesPromptOnlyScheduleYieldsNoneConfigured(t *testing.T) {
	prompt := buildOrderTaskTypesPrompt([]TaskType{
		{Key: "Schedule", Schedule: "When orders are empty"}, // case-insensitive match
	})

	if !strings.Contains(prompt, "- (none configured)") {
		t.Fatalf("prompt should fall back to none-configured when schedule is the only entry: %q", prompt)
	}
}

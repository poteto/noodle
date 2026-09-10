package loop

import (
	"strings"
	"testing"
)

func TestBuildOrderTaskTypesPromptExcludesSchedulerItself(t *testing.T) {
	prompt := buildOrderTaskTypesPrompt([]TaskType{
		{
			Key:      "schedule",
			Schedule: "When orders are empty",
		},
		{
			Key:      "execute",
			Schedule: "When work is ready",
		},
	})

	if strings.Contains(prompt, "- schedule:") {
		t.Fatalf("scheduler advertised itself: %q", prompt)
	}
	if !strings.Contains(prompt, "- execute: When work is ready") {
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

func TestBuildOrderTaskTypesPromptOnlySchedulerIsEmpty(t *testing.T) {
	prompt := buildOrderTaskTypesPrompt([]TaskType{{Key: scheduleOrderID, Schedule: "When idle"}})
	if !strings.Contains(prompt, "- (none configured)") {
		t.Fatalf("scheduler-only catalog should be empty: %q", prompt)
	}
	if strings.Contains(prompt, "- schedule:") {
		t.Fatalf("scheduler advertised itself: %q", prompt)
	}
}

package loop

import (
	"strings"
	"testing"
)

func TestBuildOrderTaskTypesPromptIncludesKeyAndSchedule(t *testing.T) {
	prompt := buildOrderTaskTypesPrompt([]TaskType{
		{
			Key:      "schedule",
			CanMerge: false,
			Schedule: "When orders are empty",
		},
	})

	if !strings.Contains(prompt, "- schedule: When orders are empty") {
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

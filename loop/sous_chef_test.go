package loop

import (
	"strings"
	"testing"
)

func TestBuildQueueTaskTypesPromptIncludesKeyAndSchedule(t *testing.T) {
	prompt := buildOrderTaskTypesPrompt([]TaskType{
		{
			Key:      "schedule",
			CanMerge: false,
			Schedule: "When the queue is empty",
		},
	})

	if !strings.Contains(prompt, "- schedule: When the queue is empty") {
		t.Fatalf("unexpected prompt: %q", prompt)
	}
}

func TestBuildQueueTaskTypesPromptEmpty(t *testing.T) {
	prompt := buildOrderTaskTypesPrompt(nil)
	if !strings.Contains(prompt, "Task types you may schedule:") {
		t.Fatalf("missing prompt header: %q", prompt)
	}
	if !strings.Contains(prompt, "- (none configured)") {
		t.Fatalf("missing empty marker: %q", prompt)
	}
}

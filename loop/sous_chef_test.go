package loop

import (
	"strings"
	"testing"
)

func TestBuildQueueTaskTypesPromptIncludesKeyAndSchedule(t *testing.T) {
	prompt := buildQueueTaskTypesPrompt([]TaskType{
		{
			Key:      "prioritize",
			CanMerge: false,
			Schedule: "When the queue is empty",
		},
	})

	if !strings.Contains(prompt, "- prioritize: When the queue is empty") {
		t.Fatalf("unexpected prompt: %q", prompt)
	}
}

func TestBuildQueueTaskTypesPromptEmpty(t *testing.T) {
	prompt := buildQueueTaskTypesPrompt(nil)
	if !strings.Contains(prompt, "Task types you may schedule:") {
		t.Fatalf("missing prompt header: %q", prompt)
	}
	if !strings.Contains(prompt, "- (none configured)") {
		t.Fatalf("missing empty marker: %q", prompt)
	}
}

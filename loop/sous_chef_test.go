package loop

import (
	"strings"
	"testing"
)

func TestBuildQueueTaskTypesPromptIncludesCanonicalFields(t *testing.T) {
	prompt := buildQueueTaskTypesPrompt([]TaskType{
		{
			Key:        "review",
			Type:       "Review",
			ConfigPath: "[review]",
			Skill:      "review",
			Blocking:   true,
			Synthetic:  true,
			Aliases:    []string{"review", "chef review"},
			Purpose:    "Blocking review gate.",
		},
	})

	if !strings.Contains(prompt, "- Review | key: review | config: [review] | skill: review | blocking: true | synthetic: true | aliases: review, chef review | purpose: Blocking review gate.") {
		t.Fatalf("unexpected prompt: %q", prompt)
	}
}

func TestBuildQueueTaskTypesPromptEmpty(t *testing.T) {
	prompt := buildQueueTaskTypesPrompt(nil)
	if !strings.Contains(prompt, "Task types you may schedule (from loop/task_types.go):") {
		t.Fatalf("missing prompt header: %q", prompt)
	}
	if !strings.Contains(prompt, "- (none configured)") {
		t.Fatalf("missing empty marker: %q", prompt)
	}
}

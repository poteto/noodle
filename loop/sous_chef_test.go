package loop

import (
	"strings"
	"testing"
)

func TestBuildQueueTaskTypesPromptIncludesKeyAndDescription(t *testing.T) {
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

	if !strings.Contains(prompt, "- review: Blocking review gate.") {
		t.Fatalf("unexpected prompt: %q", prompt)
	}
	if strings.Contains(prompt, "| config: ") || strings.Contains(prompt, "| synthetic: ") {
		t.Fatalf("expected concise prompt without verbose metadata: %q", prompt)
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

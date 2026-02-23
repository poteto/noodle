package dispatcher

import "testing"

func TestComposePromptsCodexPutsRequestBeforeSkillBundle(t *testing.T) {
	systemPrompt, finalPrompt := composePrompts(
		"codex",
		"go prompt first",
		"skill bundle second",
	)
	if systemPrompt != "" {
		t.Fatalf("system prompt = %q, want empty", systemPrompt)
	}
	if finalPrompt != "go prompt first\n\n---\n\nskill bundle second" {
		t.Fatalf("final prompt = %q", finalPrompt)
	}
}

func TestComposePromptsClaudeKeepsSystemPromptSeparate(t *testing.T) {
	systemPrompt, finalPrompt := composePrompts(
		"claude",
		"go prompt",
		"skill bundle",
	)
	if systemPrompt != "skill bundle" {
		t.Fatalf("system prompt = %q, want skill bundle", systemPrompt)
	}
	if finalPrompt != "go prompt" {
		t.Fatalf("final prompt = %q, want go prompt", finalPrompt)
	}
}

package dispatcher

import (
	"context"
	"strings"
	"testing"
)

func TestComposePromptsCodexInlinesPreambleOnly(t *testing.T) {
	systemPrompt, finalPrompt := composePrompts(
		"codex",
		"go prompt first",
		"preamble context",
		"skill bundle second",
	)
	if systemPrompt != "" {
		t.Fatalf("system prompt = %q, want empty", systemPrompt)
	}
	// Codex loads skills natively — only the preamble should be inlined.
	if finalPrompt != "go prompt first\n\n---\n\npreamble context" {
		t.Fatalf("final prompt = %q", finalPrompt)
	}
}

func TestResolveTemplateVarsUnknownPlaceholderPassesThrough(t *testing.T) {
	tmpl := "run --session={{session}} --extra={{unknown}}"
	vars := map[string]string{"session": "abc"}
	got := resolveTemplateVars(tmpl, vars)
	want := "run --session=abc --extra={{unknown}}"
	if got != want {
		t.Fatalf("resolveTemplateVars =\n  %q\nwant\n  %q", got, want)
	}
}

func TestResolveTemplateVarsEmptyTemplate(t *testing.T) {
	got := resolveTemplateVars("", map[string]string{"session": "x"})
	if got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestDispatchRejectsUnsupportedRuntimeKind(t *testing.T) {
	d := &TmuxDispatcher{}
	_, err := d.Dispatch(context.Background(), DispatchRequest{
		Name:         "cook-1",
		Prompt:       "test prompt",
		Provider:     "claude",
		Model:        "claude-sonnet-4-6",
		WorktreePath: "/tmp/worktree",
		Runtime:      "sprites",
	})
	if err == nil {
		t.Fatal("expected runtime validation error")
	}
	if !strings.Contains(err.Error(), `runtime "sprites" not configured`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestComposePromptsClaudeKeepsSystemPromptSeparate(t *testing.T) {
	systemPrompt, finalPrompt := composePrompts(
		"claude",
		"go prompt",
		"preamble",
		"skill bundle",
	)
	if systemPrompt != "preamble\n\nskill bundle" {
		t.Fatalf("system prompt = %q", systemPrompt)
	}
	if finalPrompt != "go prompt" {
		t.Fatalf("final prompt = %q, want go prompt", finalPrompt)
	}
}

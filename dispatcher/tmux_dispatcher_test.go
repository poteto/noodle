package dispatcher

import (
	"context"
	"os"
	"path/filepath"
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

func TestResolveAgentBinaryExpandsHomePath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	claudeDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatalf("mkdir claude dir: %v", err)
	}
	claudeBin := filepath.Join(claudeDir, "claude")
	if err := os.WriteFile(claudeBin, []byte(""), 0o755); err != nil {
		t.Fatalf("write claude binary stub: %v", err)
	}

	dispatcher := &TmuxDispatcher{
		providerConfigs: ProviderConfigs{
			Claude: ProviderConfig{Path: "~/.claude"},
		},
	}
	if got := dispatcher.resolveAgentBinary("claude"); got != claudeBin {
		t.Fatalf("resolveAgentBinary() = %q, want %q", got, claudeBin)
	}
}

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

func TestResolveRuntimeSelectsRequestOverDefault(t *testing.T) {
	d := &TmuxDispatcher{runtimeDefault: "default-cmd"}
	req := DispatchRequest{Runtime: "custom-cmd --repo={{repo}}"}
	got := d.resolveRuntime(req)
	if got != "custom-cmd --repo={{repo}}" {
		t.Fatalf("resolveRuntime = %q, want request runtime", got)
	}
}

func TestResolveRuntimeFallsBackToDefault(t *testing.T) {
	d := &TmuxDispatcher{runtimeDefault: "default-cmd --repo={{repo}}"}
	req := DispatchRequest{}
	got := d.resolveRuntime(req)
	if got != "default-cmd --repo={{repo}}" {
		t.Fatalf("resolveRuntime = %q, want default", got)
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

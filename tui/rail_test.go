package tui

import (
	"strings"
	"testing"
)

func TestRuntimeBadgeHiddenForLocalRuntime(t *testing.T) {
	for _, runtime := range []string{"", "tmux", "  TMUX  "} {
		if got := runtimeBadge(runtime); got != "" {
			t.Fatalf("runtimeBadge(%q) = %q, want empty", runtime, got)
		}
	}
}

func TestRuntimeBadgeShownForRemoteRuntime(t *testing.T) {
	got := runtimeBadge("sprites")
	if got == "" {
		t.Fatal("expected non-empty badge for remote runtime")
	}
	if !strings.Contains(got, "sprites") {
		t.Fatalf("badge = %q, want sprites text", got)
	}
}

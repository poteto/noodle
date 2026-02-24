package dispatcher

import "testing"

func TestNormalizeRuntimeDefaultsToTmux(t *testing.T) {
	if got := normalizeRuntime(""); got != "tmux" {
		t.Fatalf("runtime = %q, want tmux", got)
	}
	if got := normalizeRuntime("   "); got != "tmux" {
		t.Fatalf("runtime = %q, want tmux", got)
	}
}

func TestNormalizeRuntimeLowercasesAndTrims(t *testing.T) {
	if got := normalizeRuntime("  SpRiTeS "); got != "sprites" {
		t.Fatalf("runtime = %q, want sprites", got)
	}
}

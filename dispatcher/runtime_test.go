package dispatcher

import "testing"

func TestNormalizeRuntimeDefaultsToProcess(t *testing.T) {
	if got := NormalizeRuntime(""); got != "process" {
		t.Fatalf("runtime = %q, want process", got)
	}
	if got := NormalizeRuntime("   "); got != "process" {
		t.Fatalf("runtime = %q, want process", got)
	}
}

func TestNormalizeRuntimeLowercasesAndTrims(t *testing.T) {
	if got := NormalizeRuntime("  SpRiTeS "); got != "sprites" {
		t.Fatalf("runtime = %q, want sprites", got)
	}
}

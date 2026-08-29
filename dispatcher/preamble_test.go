package dispatcher

import (
	"strings"
	"testing"
)

func TestBuildSessionPreambleWorktree(t *testing.T) {
	preamble := buildSessionPreamble(false)
	if !strings.HasPrefix(preamble, "# Noodle Context") {
		t.Fatal("preamble should start with # Noodle Context")
	}
	for _, expected := range []string{
		"todos.md",
		"isolated checkout",
		".noodle/",
		"conventional commit",
	} {
		if !strings.Contains(preamble, expected) {
			t.Fatalf("preamble missing %q", expected)
		}
	}
}

func TestBuildSessionPreamblePrimaryCheckout(t *testing.T) {
	preamble := buildSessionPreamble(true)
	if !strings.HasPrefix(preamble, "# Noodle Context") {
		t.Fatal("preamble should start with # Noodle Context")
	}
	if !strings.Contains(preamble, "primary checkout") {
		t.Fatal("preamble should describe running on the primary checkout")
	}
	if !strings.Contains(preamble, "conventional commit") {
		t.Fatal("preamble should keep the shared conventions")
	}
	for _, unexpected := range []string{
		"isolated checkout",
		"isolated worktree",
		"todos.md",
		"do not modify the primary checkout",
	} {
		if strings.Contains(preamble, unexpected) {
			t.Fatalf("primary-checkout preamble should not claim %q", unexpected)
		}
	}
}

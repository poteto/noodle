package dispatcher

import (
	"strings"
	"testing"
)

func TestBuildSessionPreambleForIsolatedCook(t *testing.T) {
	preamble := buildSessionPreamble(DispatchRequest{})
	if !strings.HasPrefix(preamble, "# Noodle Context") {
		t.Fatal("preamble should start with # Noodle Context")
	}
	for _, expected := range []string{
		"todos.md",
		".noodle/",
		"conventional commit",
	} {
		if !strings.Contains(preamble, expected) {
			t.Fatalf("preamble missing %q", expected)
		}
	}
}

func TestBuildSessionPreambleForPrimaryCheckoutContainsOnlyRuntimeFacts(t *testing.T) {
	preamble := buildSessionPreamble(DispatchRequest{
		AllowPrimaryCheckout: true,
		WorktreePath:         "/tmp/project",
		Skill:                "schedule",
	})
	for _, expected := range []string{
		"working_directory: /tmp/project",
		"checkout_mode: primary-checkout",
		"selected_skill: schedule",
	} {
		if !strings.Contains(preamble, expected) {
			t.Fatalf("primary preamble missing %q: %q", expected, preamble)
		}
	}
	for _, forbidden := range []string{"todos.md", "isolated checkout", "assigned worktree", "conventional commit"} {
		if strings.Contains(preamble, forbidden) {
			t.Fatalf("primary preamble contains project policy %q: %q", forbidden, preamble)
		}
	}
}

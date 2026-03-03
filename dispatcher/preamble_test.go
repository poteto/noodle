package dispatcher

import (
	"strings"
	"testing"
)

func TestBuildSessionPreamble(t *testing.T) {
	preamble := buildSessionPreamble()
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

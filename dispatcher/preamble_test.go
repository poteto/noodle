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
		".noodle/mise.json",
		".noodle/orders.json",
		"brain/plans/",
		"brain/todos.md",
		"conventional commit",
	} {
		if !strings.Contains(preamble, expected) {
			t.Fatalf("preamble missing %q", expected)
		}
	}
}

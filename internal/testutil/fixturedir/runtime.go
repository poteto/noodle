package fixturedir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ApplyRuntimeSnapshot copies .noodle/* files from a fixture state directory
// into a temporary runtime directory. This is the shared version of the logic
// previously duplicated as applyStateRuntimeSnapshot in loop/fixture_test.go.
func ApplyRuntimeSnapshot(tb testing.TB, state FixtureState, runtimeDir string) {
	tb.Helper()
	for _, relPath := range state.FileOrder {
		normalized := filepath.ToSlash(strings.TrimSpace(relPath))
		if !strings.HasPrefix(normalized, ".noodle/") {
			continue
		}
		destRel := strings.TrimPrefix(normalized, ".noodle/")
		destination := filepath.Join(runtimeDir, filepath.FromSlash(destRel))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			tb.Fatalf("mkdir snapshot parent %s: %v", destination, err)
		}
		if err := os.WriteFile(destination, state.MustReadFile(tb, relPath), 0o644); err != nil {
			tb.Fatalf("write runtime snapshot file %s: %v", destination, err)
		}
	}
}

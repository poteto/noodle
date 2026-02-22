package fixturedir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncExpectedMarkdownUpdatesStaleFixtures(t *testing.T) {
	root := t.TempDir()
	fixtureDir := filepath.Join(root, "parse", "testdata", "sample")
	source := md(
		"---",
		"schema_version: 1",
		"expected_failure: false",
		"bug: false",
		"regression: sample",
		"---",
		"",
		"## Expected",
		"",
		"```json",
		`{"ok":true}`,
		"```",
	)
	writeFile(t, filepath.Join(fixtureDir, "expected.src.md"), source)
	writeFile(t, filepath.Join(fixtureDir, "expected.md"), strings.Replace(source, "true", "false", 1))

	updated, err := SyncExpectedMarkdown(root, false)
	if err != nil {
		t.Fatalf("sync fixtures: %v", err)
	}
	if len(updated) != 1 {
		t.Fatalf("updated count = %d", len(updated))
	}
	after, err := os.ReadFile(filepath.Join(fixtureDir, "expected.md"))
	if err != nil {
		t.Fatalf("read expected.md: %v", err)
	}
	if err := assertExpectedMarkdownSynced(
		filepath.Join(fixtureDir, "expected.src.md"),
		filepath.Join(fixtureDir, "expected.md"),
	); err != nil {
		t.Fatalf("expected.md not synced: %v\nactual:\n%s", err, string(after))
	}
}

func TestSyncExpectedMarkdownCheckMode(t *testing.T) {
	root := t.TempDir()
	fixtureDir := filepath.Join(root, "parse", "testdata", "sample")
	source := md(
		"---",
		"schema_version: 1",
		"expected_failure: false",
		"bug: false",
		"regression: sample",
		"---",
		"",
		"## Expected",
		"",
		"```json",
		`{"ok":true}`,
		"```",
	)
	writeFile(t, filepath.Join(fixtureDir, "expected.src.md"), source)
	writeFile(t, filepath.Join(fixtureDir, "expected.md"), strings.Replace(source, "true", "false", 1))

	stale, err := SyncExpectedMarkdown(root, true)
	if err == nil {
		t.Fatal("expected check mode error for stale fixture")
	}
	if len(stale) != 1 {
		t.Fatalf("stale count = %d", len(stale))
	}
}

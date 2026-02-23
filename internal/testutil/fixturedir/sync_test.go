package fixturedir

import (
	"os"
	"os/exec"
	"path/filepath"
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
		"source_hash: pending",
		"---",
		"",
		"## Expected",
		"",
		"```json",
		`{"ok":true}`,
		"```",
	)
	writeFile(t, filepath.Join(fixtureDir, "expected.md"), source)
	writeFile(t, filepath.Join(fixtureDir, "state-01", "input.ndjson"), "a\n")

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
	if err := assertExpectedMarkdownSynced(fixtureDir, filepath.Join(fixtureDir, "expected.md")); err != nil {
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
		"source_hash: pending",
		"---",
		"",
		"## Expected",
		"",
		"```json",
		`{"ok":true}`,
		"```",
	)
	writeFile(t, filepath.Join(fixtureDir, "expected.md"), source)
	writeFile(t, filepath.Join(fixtureDir, "state-01", "input.ndjson"), "a\n")
	if _, err := SyncExpectedMarkdown(root, false); err != nil {
		t.Fatalf("seed sync fixtures: %v", err)
	}
	writeFile(t, filepath.Join(fixtureDir, "state-01", "input.ndjson"), "b\n")

	stale, err := SyncExpectedMarkdown(root, true)
	if err == nil {
		t.Fatal("expected check mode error for stale fixture")
	}
	if len(stale) != 1 {
		t.Fatalf("stale count = %d", len(stale))
	}
}

func TestSyncExpectedMarkdownIgnoresGitIgnoredInputs(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	root := t.TempDir()
	fixtureDir := filepath.Join(root, "loop", "testdata", "sample")
	source := md(
		"---",
		"schema_version: 1",
		"expected_failure: false",
		"bug: false",
		"source_hash: pending",
		"---",
		"",
		"## Runtime Dump",
		"",
		"```json",
		`{"states":{}}`,
		"```",
	)
	writeFile(t, filepath.Join(root, ".gitignore"), "**/.noodle/\n")
	writeFile(t, filepath.Join(fixtureDir, "expected.md"), source)
	writeFile(t, filepath.Join(fixtureDir, "state-01", "input.json"), "{\"mise_result\":{\"backlog\":[]}}\n")

	runGit(t, root, "init")
	runGit(t, root, "add", ".")
	if _, err := SyncExpectedMarkdown(root, false); err != nil {
		t.Fatalf("seed sync fixtures: %v", err)
	}

	writeFile(t, filepath.Join(fixtureDir, "state-01", ".noodle", "queue.json"), "{}\n")
	stale, err := SyncExpectedMarkdown(root, true)
	if err != nil {
		t.Fatalf("check mode should ignore gitignored files, stale=%v err=%v", stale, err)
	}
	if len(stale) != 0 {
		t.Fatalf("stale count = %d, want 0", len(stale))
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, string(output))
	}
}

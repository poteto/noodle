package fixturedir

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverLoadsOrderedStatesAndMetadata(t *testing.T) {
	root := t.TempDir()
	alphaExpected := md(
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
		"",
		"## Expected Error",
		"",
		"```json",
		"```",
	)
	writeExpected(t, filepath.Join(root, "alpha"), alphaExpected)
	writeFile(t, filepath.Join(root, "alpha", "noodle.toml"), "[routing.defaults]\nprovider = \"claude\"\n")
	writeFile(t, filepath.Join(root, "alpha", "state-01", "input.ndjson"), "line-1\n")
	writeFile(t, filepath.Join(root, "alpha", "state-02", "noodle.toml"), "[routing.defaults]\nprovider = \"codex\"\n")
	writeFile(t, filepath.Join(root, "alpha", "state-02", ".noodle", "queue.json"), "{}\n")

	betaExpected := md(
		"---",
		"schema_version: 1",
		"expected_failure: true",
		"bug: true",
		"source_hash: pending",
		"---",
		"",
		"## Expected",
		"",
		"```json",
		`{"ok":false}`,
		"```",
	)
	writeExpected(t, filepath.Join(root, "beta"), betaExpected)
	writeFile(t, filepath.Join(root, "beta", "state-01", "input.ndjson"), "bad\n")
	syncFixtures(t, root)

	inventory, err := Discover(root)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(inventory.Cases) != 2 {
		t.Fatalf("fixture count = %d", len(inventory.Cases))
	}
	if strings.Join(inventory.Names(), ",") != "alpha,beta" {
		t.Fatalf("fixture names = %v", inventory.Names())
	}

	alpha := inventory.Cases[0]
	if alpha.Metadata.SchemaVersion != FixtureSchemaVersion {
		t.Fatalf("schema version = %d", alpha.Metadata.SchemaVersion)
	}
	if alpha.Metadata.ExpectedFailure {
		t.Fatal("alpha expected_failure should be false")
	}
	if alpha.Metadata.Bug {
		t.Fatal("alpha bug should be false")
	}
	if len(alpha.States) != 2 {
		t.Fatalf("state count = %d", len(alpha.States))
	}
	if alpha.States[0].ID != "state-01" || alpha.States[1].ID != "state-02" {
		t.Fatalf("state ordering = %#v", []string{alpha.States[0].ID, alpha.States[1].ID})
	}
	if alpha.States[1].ConfigScope.StateOverridePath == "" {
		t.Fatal("missing state override path")
	}
	if alpha.States[0].ConfigScope.BaseConfigPath == "" {
		t.Fatal("missing base config path")
	}
	if _, ok := alpha.Section("Expected"); !ok {
		t.Fatal("missing expected section")
	}
	if alpha.ExpectedError != nil {
		t.Fatalf("alpha expected error = %#v", alpha.ExpectedError)
	}

	beta := inventory.Cases[1]
	if !beta.Metadata.ExpectedFailure {
		t.Fatal("beta expected_failure should be true")
	}
	if !beta.Metadata.Bug {
		t.Fatal("beta bug should be true")
	}
	if beta.ExpectedError == nil || !beta.ExpectedError.Any {
		t.Fatalf("beta expected error = %#v", beta.ExpectedError)
	}
}

func TestDiscoverRejectsUnsupportedFrontmatterKey(t *testing.T) {
	root := t.TempDir()
	content := md(
		"---",
		"schema_version: 1",
		"expected_failure: false",
		"bug: false",
		"source_hash: pending",
		"owner: tooling",
		"---",
		"",
		"## Expected",
		"",
		"```json",
		"{}",
		"```",
	)
	writeExpected(t, filepath.Join(root, "bad-frontmatter"), content)
	writeFile(t, filepath.Join(root, "bad-frontmatter", "state-01", "input.ndjson"), "ok\n")

	_, err := Discover(root)
	if err == nil || !strings.Contains(err.Error(), "unsupported frontmatter key") {
		t.Fatalf("discover err = %v", err)
	}
}

func TestDiscoverRejectsSchemaVersionMismatch(t *testing.T) {
	root := t.TempDir()
	content := md(
		"---",
		"schema_version: 9",
		"expected_failure: false",
		"bug: false",
		"source_hash: pending",
		"---",
		"",
		"## Expected",
		"",
		"```json",
		"{}",
		"```",
	)
	writeExpected(t, filepath.Join(root, "schema-mismatch"), content)
	writeFile(t, filepath.Join(root, "schema-mismatch", "state-01", "input.ndjson"), "ok\n")

	_, err := Discover(root)
	if err == nil || !strings.Contains(err.Error(), "unsupported schema_version") {
		t.Fatalf("discover err = %v", err)
	}
}

func TestDiscoverRejectsBugWithoutExpectedFailure(t *testing.T) {
	root := t.TempDir()
	content := md(
		"---",
		"schema_version: 1",
		"expected_failure: false",
		"bug: true",
		"source_hash: pending",
		"---",
		"",
		"## Expected",
		"",
		"```json",
		"{}",
		"```",
	)
	writeExpected(t, filepath.Join(root, "bad-bug-flag"), content)
	writeFile(t, filepath.Join(root, "bad-bug-flag", "state-01", "input.ndjson"), "ok\n")

	_, err := Discover(root)
	if err == nil || !strings.Contains(err.Error(), "bug=true requires expected_failure=true") {
		t.Fatalf("discover err = %v", err)
	}
}

func TestValidateFixtureRootDetectsLayoutIssues(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "missing-expected", "state-01", "input.ndjson"), "x\n")
	writeExpected(t, filepath.Join(root, "gap-states"), md(
		"---",
		"schema_version: 1",
		"expected_failure: false",
		"bug: false",
		"source_hash: pending",
		"---",
	))
	writeFile(t, filepath.Join(root, "gap-states", "state-01", "input.ndjson"), "x\n")
	writeFile(t, filepath.Join(root, "gap-states", "state-03", "input.ndjson"), "x\n")

	issues, err := ValidateFixtureRoot(root)
	if err != nil {
		t.Fatalf("validate fixture root: %v", err)
	}
	if len(issues) == 0 {
		t.Fatal("expected validation issues")
	}

	messages := make([]string, 0, len(issues))
	for _, issue := range issues {
		messages = append(messages, issue.Message)
	}
	joined := strings.Join(messages, " | ")
	if !strings.Contains(joined, "missing required expected.md") {
		t.Fatalf("issues = %s", joined)
	}
	if !strings.Contains(joined, "state ordering gap") {
		t.Fatalf("issues = %s", joined)
	}
}

func TestDiscoverRejectsUnknownExpectedErrorKeys(t *testing.T) {
	root := t.TempDir()
	writeExpected(t, filepath.Join(root, "bad-error"), md(
		"---",
		"schema_version: 1",
		"expected_failure: true",
		"bug: true",
		"source_hash: pending",
		"---",
		"",
		"## Expected",
		"",
		"```json",
		"{}",
		"```",
		"",
		"## Expected Error",
		"",
		"```json",
		`{"nope":true}`,
		"```",
	))
	writeFile(t, filepath.Join(root, "bad-error", "state-01", "input.ndjson"), "x\n")
	syncFixtures(t, root)

	_, err := Discover(root)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("discover err = %v", err)
	}
}

func TestDiscoverRejectsOutOfDateExpectedMarkdown(t *testing.T) {
	root := t.TempDir()
	writeExpected(t, filepath.Join(root, "stale"), md(
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
	))
	writeFile(t, filepath.Join(root, "stale", "state-01", "input.ndjson"), "x\n")
	syncFixtures(t, root)
	writeFile(t, filepath.Join(root, "stale", "state-01", "input.ndjson"), "changed\n")

	_, err := Discover(root)
	if err == nil || !strings.Contains(err.Error(), "expected.md is out of date") {
		t.Fatalf("discover err = %v", err)
	}
}

func TestDiscoverIgnoresGitIgnoredStateRuntimeArtifacts(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "**/.noodle/\n")
	writeExpected(t, filepath.Join(root, "sample"), md(
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
	))
	writeFile(t, filepath.Join(root, "sample", "state-01", "input.json"), "{\"mise_result\":{\"backlog\":[]}}\n")

	runGit(t, root, "init")
	runGit(t, root, "add", ".")
	syncFixtures(t, root)

	writeFile(t, filepath.Join(root, "sample", "state-01", ".noodle", "queue.json"), "{\"items\":[]}\n")

	inventory, err := Discover(root)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	state := inventory.Cases[0].States[0]
	for _, rel := range state.FileOrder {
		if strings.Contains(filepath.ToSlash(rel), ".noodle/") {
			t.Fatalf("unexpected gitignored runtime file in state file order: %s", rel)
		}
	}
}

func md(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeExpected(t *testing.T, fixtureDir, content string) {
	t.Helper()
	writeFile(t, filepath.Join(fixtureDir, "expected.md"), content)
}

func syncFixtures(t *testing.T, root string) {
	t.Helper()
	updated, err := SyncExpectedMarkdown(root, false)
	if err != nil {
		t.Fatalf("sync fixtures: %v", err)
	}
	if len(updated) == 0 {
		t.Fatal("expected fixture sync to update source_hash")
	}
}

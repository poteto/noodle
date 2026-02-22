package fixturedir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverLoadsOrderedStatesAndMetadata(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "alpha", "expected.md"), md(
		"---",
		"schema_version: 1",
		"expected_failure: false",
		"bug: alpha-happy-path",
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
	))
	writeFile(t, filepath.Join(root, "alpha", "noodle.toml"), "[routing.defaults]\nprovider = \"claude\"\n")
	writeFile(t, filepath.Join(root, "alpha", "state-01", "input.ndjson"), "line-1\n")
	writeFile(t, filepath.Join(root, "alpha", "state-02", "noodle.toml"), "[routing.defaults]\nprovider = \"codex\"\n")
	writeFile(t, filepath.Join(root, "alpha", "state-02", ".noodle", "queue.json"), "{}\n")

	writeFile(t, filepath.Join(root, "beta", "expected.md"), md(
		"---",
		"schema_version: 1",
		"expected_failure: true",
		"bug: beta-failure",
		"---",
		"",
		"## Expected",
		"",
		"```json",
		`{"ok":false}`,
		"```",
	))
	writeFile(t, filepath.Join(root, "beta", "state-01", "input.ndjson"), "bad\n")

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
	if alpha.Metadata.Bug != "alpha-happy-path" {
		t.Fatalf("bug = %q", alpha.Metadata.Bug)
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
	if beta.ExpectedError == nil || !beta.ExpectedError.Any {
		t.Fatalf("beta expected error = %#v", beta.ExpectedError)
	}
}

func TestDiscoverRejectsUnsupportedFrontmatterKey(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "bad-frontmatter", "expected.md"), md(
		"---",
		"schema_version: 1",
		"expected_failure: false",
		"bug: bad-fixture",
		"owner: tooling",
		"---",
		"",
		"## Expected",
		"",
		"```json",
		"{}",
		"```",
	))
	writeFile(t, filepath.Join(root, "bad-frontmatter", "state-01", "input.ndjson"), "ok\n")

	_, err := Discover(root)
	if err == nil || !strings.Contains(err.Error(), "unsupported frontmatter key") {
		t.Fatalf("discover err = %v", err)
	}
}

func TestDiscoverRejectsSchemaVersionMismatch(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "schema-mismatch", "expected.md"), md(
		"---",
		"schema_version: 9",
		"expected_failure: false",
		"bug: mismatch",
		"---",
		"",
		"## Expected",
		"",
		"```json",
		"{}",
		"```",
	))
	writeFile(t, filepath.Join(root, "schema-mismatch", "state-01", "input.ndjson"), "ok\n")

	_, err := Discover(root)
	if err == nil || !strings.Contains(err.Error(), "unsupported schema_version") {
		t.Fatalf("discover err = %v", err)
	}
}

func TestValidateFixtureRootDetectsLayoutIssues(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "missing-expected", "state-01", "input.ndjson"), "x\n")
	writeFile(t, filepath.Join(root, "gap-states", "expected.md"), md(
		"---",
		"schema_version: 1",
		"expected_failure: false",
		"bug: gap",
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
	writeFile(t, filepath.Join(root, "bad-error", "expected.md"), md(
		"---",
		"schema_version: 1",
		"expected_failure: true",
		"bug: bad-error-expectation",
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

	_, err := Discover(root)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("discover err = %v", err)
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

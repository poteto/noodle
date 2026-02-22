package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunFixturesCommandRequiresSubcommand(t *testing.T) {
	err := runFixturesCommand(context.Background(), nil, nil, nil)
	if err == nil {
		t.Fatal("expected missing subcommand error")
	}
	if !strings.Contains(err.Error(), "subcommand is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunFixturesCommandSync(t *testing.T) {
	root := t.TempDir()
	fixtureDir := filepath.Join(root, "parse", "testdata", "sample")
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}

	source := strings.Join([]string{
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
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(fixtureDir, "expected.src.md"), []byte(source), 0o644); err != nil {
		t.Fatalf("write expected.src.md: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(fixtureDir, "expected.md"),
		[]byte(strings.Replace(source, "true", "false", 1)),
		0o644,
	); err != nil {
		t.Fatalf("write expected.md: %v", err)
	}

	output := captureStdout(t, func() {
		err := runFixturesCommand(context.Background(), nil, nil, []string{"sync", "--root", root})
		if err != nil {
			t.Fatalf("runFixturesCommand sync: %v", err)
		}
	})
	if !strings.Contains(output, "fixtures sync: updated 1 file") {
		t.Fatalf("unexpected output: %q", output)
	}

	after, err := os.ReadFile(filepath.Join(fixtureDir, "expected.md"))
	if err != nil {
		t.Fatalf("read expected.md: %v", err)
	}
	if !strings.Contains(string(after), "source_hash:") {
		t.Fatalf("expected generated source_hash in expected.md:\n%s", string(after))
	}
	if strings.Contains(string(after), `{"ok":false}`) {
		t.Fatalf("expected.md not synced:\n%s", string(after))
	}
}

func TestRunFixturesCommandCheck(t *testing.T) {
	root := t.TempDir()
	fixtureDir := filepath.Join(root, "parse", "testdata", "sample")
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}

	source := strings.Join([]string{
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
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(fixtureDir, "expected.src.md"), []byte(source), 0o644); err != nil {
		t.Fatalf("write expected.src.md: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(fixtureDir, "expected.md"),
		[]byte(strings.Replace(source, "true", "false", 1)),
		0o644,
	); err != nil {
		t.Fatalf("write expected.md: %v", err)
	}

	err := runFixturesCommand(context.Background(), nil, nil, []string{"check", "--root", root})
	if err == nil {
		t.Fatal("expected stale fixture check failure")
	}
	if !strings.Contains(err.Error(), "out of date") {
		t.Fatalf("unexpected error: %v", err)
	}
}

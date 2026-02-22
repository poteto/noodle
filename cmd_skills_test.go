package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poteto/noodle/config"
)

func TestRunSkillsCommandRequiresSubcommand(t *testing.T) {
	app := &App{Config: config.DefaultConfig()}

	err := runSkillsCommand(context.Background(), app, nil, nil)
	if err == nil {
		t.Fatal("expected missing subcommand error")
	}
	if !strings.Contains(err.Error(), "subcommand is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunSkillsListCommandRespectsPrecedence(t *testing.T) {
	project := t.TempDir()
	user := t.TempDir()

	mustMkdirAll(t, filepath.Join(project, "review"))
	mustWriteFile(t, filepath.Join(project, "review", "SKILL.md"), "# review")
	mustMkdirAll(t, filepath.Join(user, "review"))
	mustMkdirAll(t, filepath.Join(user, "debug"))
	mustWriteFile(t, filepath.Join(user, "debug", "SKILL.md"), "# debug")

	app := &App{
		Config: config.Config{
			Skills: config.SkillsConfig{Paths: []string{project, user}},
		},
	}

	output := captureStdout(t, func() {
		err := runSkillsCommand(context.Background(), app, nil, []string{"list"})
		if err != nil {
			t.Fatalf("runSkillsCommand: %v", err)
		}
	})

	if !strings.Contains(output, "review\t"+project+"\ttrue\t") {
		t.Fatalf("expected project review skill in output: %q", output)
	}
	if !strings.Contains(output, "debug\t"+user+"\ttrue\t") {
		t.Fatalf("expected user debug skill in output: %q", output)
	}
	if strings.Contains(output, "review\t"+user+"\t") {
		t.Fatalf("expected user review skill to be shadowed: %q", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = originalStdout
	})

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close write pipe: %v", err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close read pipe: %v", err)
	}
	return string(out)
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

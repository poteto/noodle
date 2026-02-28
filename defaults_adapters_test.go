package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultBacklogSyncParsesTodos(t *testing.T) {
	root := t.TempDir()
	todos := filepath.Join(root, "brain", "todos.md")
	if err := os.MkdirAll(filepath.Dir(todos), 0o755); err != nil {
		t.Fatalf("mkdir todos dir: %v", err)
	}
	content := `# Todos

<!-- next-id: 5 -->

## Frontend

1. [ ] Fix login bug #auth ~medium [[plans/03-auth-refactor/overview]]
2. [x] Update docs #docs
`
	if err := os.WriteFile(todos, []byte(content), 0o644); err != nil {
		t.Fatalf("write todos: %v", err)
	}

	script := filepath.Join("defaults", "adapters", "backlog-sync")
	cmd := exec.Command(script)
	cmd.Env = append(os.Environ(), "NOODLE_TODOS_FILE="+todos)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("run backlog-sync: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 {
		t.Fatalf("line count = %d", len(lines))
	}
	var item map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &item); err != nil {
		t.Fatalf("parse line 1: %v", err)
	}
	if item["id"] != "1" || item["status"] != "open" {
		t.Fatalf("unexpected item: %#v", item)
	}
	if item["plan"] != "brain/plans/03-auth-refactor/overview.md" {
		t.Fatalf("plan = %v, want brain/plans/03-auth-refactor/overview.md", item["plan"])
	}
}

func TestDefaultBacklogAddDoneEditRoundTrip(t *testing.T) {
	root := t.TempDir()
	todos := filepath.Join(root, "brain", "todos.md")
	if err := os.MkdirAll(filepath.Dir(todos), 0o755); err != nil {
		t.Fatalf("mkdir todos dir: %v", err)
	}
	if err := os.WriteFile(todos, []byte("# Todos\n\n<!-- next-id: 1 -->\n\n## Inbox\n"), 0o644); err != nil {
		t.Fatalf("write todos: %v", err)
	}

	run := func(name string, args ...string) {
		t.Helper()
		script := filepath.Join("defaults", "adapters", name)
		cmd := exec.Command(script, args...)
		cmd.Env = append(os.Environ(), "NOODLE_TODOS_FILE="+todos)
		if strings.Contains(name, "add") || strings.Contains(name, "edit") {
			if strings.Contains(name, "add") {
				cmd.Stdin = strings.NewReader(`{"title":"Ship feature"}`)
			} else {
				cmd.Stdin = strings.NewReader(`{"title":"Ship feature v2"}`)
			}
		}
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s failed: %v\n%s", name, err, out)
		}
	}

	run("backlog-add")
	run("backlog-edit", "1")
	run("backlog-done", "1")

	data, err := os.ReadFile(todos)
	if err != nil {
		t.Fatalf("read todos: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "1. [x] Ship feature v2") {
		t.Fatalf("unexpected todos content:\n%s", text)
	}
}


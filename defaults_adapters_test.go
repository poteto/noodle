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
	cmd.Dir = "/Users/lauren/code/noodle/.worktrees/phase-09-scheduling-loop"
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
		cmd.Dir = "/Users/lauren/code/noodle/.worktrees/phase-09-scheduling-loop"
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

func TestDefaultPlansCreateSyncAndPhaseAdd(t *testing.T) {
	root := t.TempDir()
	plansDir := filepath.Join(root, "brain", "plans")
	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		t.Fatalf("mkdir plans dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(plansDir, "index.md"), []byte("# Plans\n\n"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	run := func(name string, stdin string, args ...string) string {
		t.Helper()
		script := filepath.Join("defaults", "adapters", name)
		cmd := exec.Command(script, args...)
		cmd.Dir = "/Users/lauren/code/noodle/.worktrees/phase-09-scheduling-loop"
		cmd.Env = append(os.Environ(), "NOODLE_PLANS_DIR="+plansDir)
		if stdin != "" {
			cmd.Stdin = strings.NewReader(stdin)
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s failed: %v\n%s", name, err, out)
		}
		return strings.TrimSpace(string(out))
	}

	id := run("plan-create", `{"title":"Auth Refactor","slug":"auth-refactor"}`)
	if id != "01" {
		t.Fatalf("plan id = %q", id)
	}
	run("plan-phase-add", `{"name":"Implement"}`, "1")

	synced := run("plans-sync", "")
	if synced == "" {
		t.Fatal("expected plans-sync output")
	}
	line := strings.Split(strings.TrimSpace(synced), "\n")[0]
	var item map[string]any
	if err := json.Unmarshal([]byte(line), &item); err != nil {
		t.Fatalf("parse plans-sync output: %v", err)
	}
	if item["id"] != "01" {
		t.Fatalf("unexpected plan id: %#v", item)
	}
}

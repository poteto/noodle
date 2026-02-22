package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poteto/noodle/config"
)

func TestRunDebugCommandDoesNotAcceptArguments(t *testing.T) {
	err := runDebugCommand(context.Background(), nil, nil, []string{"extra"})
	if err == nil {
		t.Fatal("expected argument rejection")
	}
	if !strings.Contains(err.Error(), "does not accept arguments") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDebugCommandOutputsDeterministicDump(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions", "cook-b"), 0o755); err != nil {
		t.Fatalf("mkdir cook-b: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions", "cook-a"), 0o755); err != nil {
		t.Fatalf("mkdir cook-a: %v", err)
	}
	write := func(path string, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	write(
		filepath.Join(runtimeDir, "queue.json"),
		`{"items":[{"id":"2","provider":"claude"},{"id":"1","provider":"codex","model":"gpt-5.3-codex"}]}`,
	)
	write(
		filepath.Join(runtimeDir, "sessions", "cook-a", "meta.json"),
		`{"status":"running","loop_state":"paused","target":"10","total_cost_usd":1.25}`,
	)
	write(
		filepath.Join(runtimeDir, "sessions", "cook-a", "spawn.json"),
		`{"provider":"codex","model":"gpt-5.3-codex"}`,
	)
	write(
		filepath.Join(runtimeDir, "sessions", "cook-b", "meta.json"),
		`{"status":"exited","loop_state":"draining","target":"11","total_cost_usd":0.75}`,
	)
	write(filepath.Join(runtimeDir, "failed.json"), `{"11":"merge failed"}`)
	write(filepath.Join(runtimeDir, "control-ack.ndjson"), "{\"id\":\"1\"}\n{\"id\":\"2\"}\n")

	cfg := config.DefaultConfig()
	cfg.Routing.Defaults.Provider = "codex"
	cfg.Routing.Defaults.Model = "gpt-5.3-codex"
	cfg.Phases = map[string]string{"debugging": "oops"}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir project dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	run := func() string {
		return captureStdout(t, func() {
			if err := runDebugCommand(context.Background(), &App{Config: cfg}, nil, nil); err != nil {
				t.Fatalf("runDebugCommand: %v", err)
			}
		})
	}

	first := run()
	second := run()
	if first != second {
		t.Fatalf("debug output is not deterministic\nfirst:\n%s\nsecond:\n%s", first, second)
	}

	var decoded debugDump
	if err := json.Unmarshal([]byte(first), &decoded); err != nil {
		t.Fatalf("parse debug output: %v\n%s", err, first)
	}
	if decoded.SchemaVersion != 1 {
		t.Fatalf("schema_version = %d", decoded.SchemaVersion)
	}
	if decoded.Runtime.LoopState != "draining" {
		t.Fatalf("loop_state = %q", decoded.Runtime.LoopState)
	}
	if len(decoded.Runtime.Sessions) != 2 {
		t.Fatalf("sessions = %d", len(decoded.Runtime.Sessions))
	}
	if decoded.Runtime.Sessions[0].ID != "cook-a" {
		t.Fatalf("first session ID = %q", decoded.Runtime.Sessions[0].ID)
	}
	if decoded.Runtime.ControlAckCount != 2 {
		t.Fatalf("control_ack_count = %d", decoded.Runtime.ControlAckCount)
	}
	if decoded.Runtime.FailedTargets["11"] != "merge failed" {
		t.Fatalf("failed target reason = %q", decoded.Runtime.FailedTargets["11"])
	}
}

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunStatusNoActiveCooks(t *testing.T) {
	projectDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir project dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})

	output := captureStdout(t, func() {
		if err := runStatus(&App{}); err != nil {
			t.Fatalf("runStatus: %v", err)
		}
	})

	if !strings.Contains(output, "no active cooks") {
		t.Fatalf("expected no-active output, got: %q", output)
	}
	if !strings.Contains(output, "queue=0") {
		t.Fatalf("expected queue depth output, got: %q", output)
	}
	if !strings.Contains(output, "cost=$0.00") {
		t.Fatalf("expected total cost output, got: %q", output)
	}
}

func TestRunStatusReadsSessionsAndQueue(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")

	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions", "cook-a"), 0o755); err != nil {
		t.Fatalf("mkdir cook-a: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions", "cook-b"), 0o755); err != nil {
		t.Fatalf("mkdir cook-b: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, "sessions", "cook-a", "meta.json"),
		[]byte(`{"status":"running","total_cost_usd":1.25,"loop_state":"paused"}`),
		0o644,
	); err != nil {
		t.Fatalf("write cook-a meta: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, "sessions", "cook-b", "meta.json"),
		[]byte(`{"status":"exited","total_cost_usd":0.75,"loop_state":"draining"}`),
		0o644,
	); err != nil {
		t.Fatalf("write cook-b meta: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, "orders.json"),
		[]byte(`{"orders":[{"id":"1","stages":[{"status":"pending","provider":"c","model":"m"}],"status":"active"},{"id":"2","stages":[{"status":"pending","provider":"c","model":"m"}],"status":"active"},{"id":"3","stages":[{"status":"pending","provider":"c","model":"m"}],"status":"active"}]}`),
		0o644,
	); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir project dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})

	output := captureStdout(t, func() {
		if err := runStatus(&App{}); err != nil {
			t.Fatalf("runStatus: %v", err)
		}
	})

	if !strings.Contains(output, "active cooks=1") {
		t.Fatalf("expected active cook count, got: %q", output)
	}
	if !strings.Contains(output, "queue=3") {
		t.Fatalf("expected queue depth, got: %q", output)
	}
	if !strings.Contains(output, "cost=$2.00") {
		t.Fatalf("expected total cost, got: %q", output)
	}
	if !strings.Contains(output, "loop=draining") {
		t.Fatalf("expected loop state, got: %q", output)
	}
}

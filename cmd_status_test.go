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
	if !strings.Contains(output, "orders=0") {
		t.Fatalf("expected orders depth output, got: %q", output)
	}
}

func TestRunStatusReadsSessionsAndOrders(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")

	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, "status.json"),
		[]byte(`{"active":["order-1"],"loop_state":"draining","max_concurrency":4}`),
		0o644,
	); err != nil {
		t.Fatalf("write status: %v", err)
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
	if !strings.Contains(output, "orders=3") {
		t.Fatalf("expected orders depth, got: %q", output)
	}
	if !strings.Contains(output, "loop=draining") {
		t.Fatalf("expected loop state, got: %q", output)
	}
}

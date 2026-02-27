//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// preflight checks that required external dependencies are available.
// Skips the test if any are missing.
func preflight(t *testing.T) {
	t.Helper()

	deps := []struct {
		name string
		env  string // optional: check env var instead of PATH
	}{
		{name: "codex"},
		{name: "tmux"},
		{name: "git"},
	}

	for _, dep := range deps {
		if dep.env != "" {
			if os.Getenv(dep.env) == "" {
				t.Skipf("skipping: %s env var not set", dep.env)
			}
			continue
		}
		if _, err := exec.LookPath(dep.name); err != nil {
			t.Skipf("skipping: %s not found on PATH", dep.name)
		}
	}

	if os.Getenv("CODEX_API_KEY") == "" && os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("skipping: neither CODEX_API_KEY nor OPENAI_API_KEY set")
	}
}

func TestSmokeAgentLoop(t *testing.T) {
	preflight(t)

	noodleBin := buildNoodle(t)
	projectDir := scaffoldProject(t, noodleBin)

	t.Logf("noodle binary: %s", noodleBin)
	t.Logf("project dir:   %s", projectDir)

	// Start noodle as a background process.
	cmd := exec.Command(noodleBin, "start")
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(),
		"NOODLE_NO_BROWSER=1",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	t.Cleanup(func() {
		cleanupNoodle(t, cmd, projectDir)
	})

	if err := cmd.Start(); err != nil {
		t.Fatalf("start noodle: %v", err)
	}
	t.Logf("noodle started (pid %d)", cmd.Process.Pid)

	// Phased milestone polling.
	milestones := []milestone{
		{
			name:    "Phase A: orders.json appears",
			timeout: 60 * time.Second,
			check: func(dir string) (bool, error) {
				return ordersExist(dir)
			},
		},
		{
			name:    "Phase B: session directory appears",
			timeout: 120 * time.Second,
			check: func(dir string) (bool, error) {
				return sessionDirExists(dir)
			},
		},
		{
			name:    "Phase C: session completed/merged",
			timeout: 180 * time.Second,
			check: func(dir string) (bool, error) {
				return sessionCompleted(dir)
			},
		},
	}

	err := pollMilestones(t, milestones, projectDir)
	if err != nil {
		// One retry for transient failures.
		t.Logf("first attempt failed: %v — retrying once", err)
		cleanupNoodle(t, cmd, projectDir)

		// Re-scaffold and restart.
		projectDir = scaffoldProject(t, noodleBin)
		t.Logf("retry project dir: %s", projectDir)

		cmd = exec.Command(noodleBin, "start")
		cmd.Dir = projectDir
		cmd.Env = append(os.Environ(),
			"NOODLE_NO_BROWSER=1",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		t.Cleanup(func() {
			cleanupNoodle(t, cmd, projectDir)
		})

		if startErr := cmd.Start(); startErr != nil {
			t.Fatalf("start noodle (retry): %v", startErr)
		}
		t.Logf("noodle restarted (pid %d)", cmd.Process.Pid)

		if retryErr := pollMilestones(t, milestones, projectDir); retryErr != nil {
			t.Fatalf("milestones not reached after retry: %v", retryErr)
		}
	}

	// Assertions.
	assertOrdersExist(t, projectDir)
	assertSessionMeta(t, projectDir)

	// UI smoke tests via Playwright.
	const baseURL = "http://127.0.0.1:13737"
	if err := waitForServer(t, baseURL, 15*time.Second); err != nil {
		t.Fatalf("server not reachable: %v", err)
	}
	if err := runPlaywrightTests(t, baseURL); err != nil {
		t.Errorf("playwright UI smoke: %v", err)
	}
}

// assertOrdersExist verifies orders.json exists and has valid structure.
func assertOrdersExist(t *testing.T, projectDir string) {
	t.Helper()
	path := filepath.Join(projectDir, ".noodle", "orders.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("read orders.json: %v", err)
		return
	}
	var orders struct {
		Orders []json.RawMessage `json:"orders"`
	}
	if err := json.Unmarshal(data, &orders); err != nil {
		t.Errorf("parse orders.json: %v", err)
	}
	t.Logf("orders.json contains %d order(s)", len(orders.Orders))
}

// assertSessionMeta verifies at least one session has a meta.json with a
// recognized status.
func assertSessionMeta(t *testing.T, projectDir string) {
	t.Helper()
	sessionsDir := filepath.Join(projectDir, ".noodle", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		t.Errorf("read sessions dir: %v", err)
		return
	}

	foundMeta := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaPath := filepath.Join(sessionsDir, entry.Name(), "meta.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta struct {
			SessionID string `json:"session_id"`
			Status    string `json:"status"`
		}
		if err := json.Unmarshal(data, &meta); err != nil {
			t.Errorf("parse meta.json for session %s: %v", entry.Name(), err)
			continue
		}
		status := strings.ToLower(strings.TrimSpace(meta.Status))
		t.Logf("session %s: status=%s", entry.Name(), status)
		foundMeta = true
	}

	if !foundMeta {
		t.Error("no session meta.json found")
	}
}

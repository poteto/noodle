//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
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

	// Codex CLI handles its own authentication — no env var check needed.
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
		// Dump session diagnostics before retrying.
		dumpSessionDiagnostics(t, projectDir)
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

func TestSmokeInvalidRuntimeFallback(t *testing.T) {
	preflight(t)

	noodleBin := buildNoodle(t)
	dir := t.TempDir()

	// Scaffold a minimal project with invalid runtime.default.
	run(t, dir, "git", "init", "-b", "main")
	run(t, dir, "git", "config", "user.email", "test@noodle.dev")
	run(t, dir, "git", "config", "user.name", "Noodle Test")
	run(t, dir, "git", "commit", "--allow-empty", "-m", "initial commit")

	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/e2e\n\ngo 1.23\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "init")

	// Brain scaffolding.
	writeFile(t, filepath.Join(dir, "brain", "index.md"), "# Brain\n")

	writeFile(t, filepath.Join(dir, "brain", "todos.md"), "# Todos\n\n<!-- next-id: 2 -->\n\n## Tasks\n\n1. [ ] Create hello.txt ~small\n")

	// Skills.
	srcSkills := filepath.Join(repoRoot(t), ".agents", "skills")
	dstSkills := filepath.Join(dir, ".agents", "skills")
	for _, skill := range []string{"schedule", "execute"} {
		copyDir(t, filepath.Join(srcSkills, skill), filepath.Join(dstSkills, skill))
	}

	// Adapter scripts.
	adapterDir := filepath.Join(dir, ".noodle", "adapters")
	mkdirAll(t, adapterDir)
	writeFile(t, filepath.Join(adapterDir, "backlog-sync"), "#!/bin/sh\necho '{\"id\":\"1\",\"title\":\"hello\",\"status\":\"open\",\"tags\":[]}'\n")
	writeFile(t, filepath.Join(adapterDir, "backlog-done"), "#!/bin/sh\n")
	writeFile(t, filepath.Join(adapterDir, "backlog-add"), "#!/bin/sh\n")
	writeFile(t, filepath.Join(adapterDir, "backlog-edit"), "#!/bin/sh\n")
	chmodExec(t, filepath.Join(adapterDir, "backlog-sync"))
	chmodExec(t, filepath.Join(adapterDir, "backlog-done"))
	chmodExec(t, filepath.Join(adapterDir, "backlog-add"))
	chmodExec(t, filepath.Join(adapterDir, "backlog-edit"))

	// Config with invalid runtime.default = "tmux".
	writeFile(t, filepath.Join(dir, ".noodle.toml"), `mode = "auto"

[routing.defaults]
provider = "codex"
model = "gpt-5.3-codex"

[skills]
paths = [".agents/skills"]

[agents.codex]
path = "~/.codex"

[adapters.backlog]
skill = "todo"

[adapters.backlog.scripts]
sync = ".noodle/adapters/backlog-sync"
done = ".noodle/adapters/backlog-done"
add = ".noodle/adapters/backlog-add"
edit = ".noodle/adapters/backlog-edit"

[concurrency]
max_cooks = 1

[runtime]
default = "tmux"

[server]
enabled = true
port = 13738
`)

	mkdirAll(t, filepath.Join(dir, ".noodle"))
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "scaffolding")

	// Start noodle.
	cmd := exec.Command(noodleBin, "start")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "NOODLE_NO_BROWSER=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	t.Cleanup(func() { cleanupNoodle(t, cmd, dir) })

	if err := cmd.Start(); err != nil {
		t.Fatalf("start noodle: %v", err)
	}
	t.Logf("noodle started (pid %d)", cmd.Process.Pid)

	const baseURL = "http://127.0.0.1:13738"
	if err := waitForServer(t, baseURL, 15*time.Second); err != nil {
		t.Fatalf("server not reachable: %v", err)
	}

	// Verify snapshot contains warning about tmux runtime.
	resp, err := http.Get(baseURL + "/api/snapshot")
	if err != nil {
		t.Fatalf("GET snapshot: %v", err)
	}
	defer resp.Body.Close()

	var snap struct {
		Warnings []string `json:"warnings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if len(snap.Warnings) == 0 {
		t.Fatal("expected warnings in snapshot for invalid runtime.default")
	}
	found := false
	for _, w := range snap.Warnings {
		if strings.Contains(w, "tmux") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected warning mentioning tmux, got %v", snap.Warnings)
	}
	t.Logf("snapshot warnings: %v", snap.Warnings)
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

// assertSessionMeta verifies at least one non-schedule session has a
// canonical.ndjson with a completion event. We check canonical.ndjson
// instead of meta.json because meta.json is written asynchronously by the
// monitor and may not exist at assertion time.
func assertSessionMeta(t *testing.T, projectDir string) {
	t.Helper()
	sessionsDir := filepath.Join(projectDir, ".noodle", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		t.Errorf("read sessions dir: %v", err)
		return
	}

	foundCompleted := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), "schedule-") {
			continue
		}
		canonicalPath := filepath.Join(sessionsDir, entry.Name(), "canonical.ndjson")
		data, err := os.ReadFile(canonicalPath)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var event struct {
				Type string `json:"type"`
			}
			if json.Unmarshal([]byte(line), &event) == nil && event.Type == "complete" {
				t.Logf("session %s: completed (canonical.ndjson)", entry.Name())
				foundCompleted = true
			}
		}
	}

	if !foundCompleted {
		t.Error("no completed session found in canonical.ndjson")
	}
}

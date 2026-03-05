//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
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
	configureProcessGroup(cmd)
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
	preUISmokeMilestones := []milestone{
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
	}
	postUISmokeMilestones := []milestone{
		{
			name:    "Phase C: session completed/merged",
			timeout: 180 * time.Second,
			check: func(dir string) (bool, error) {
				return sessionCompleted(dir)
			},
		},
	}

	const baseURL = "http://127.0.0.1:13737"
	runSmokePhases := func() error {
		if err := pollMilestones(t, preUISmokeMilestones, projectDir); err != nil {
			return err
		}
		if err := waitForServer(t, baseURL, 15*time.Second); err != nil {
			return fmt.Errorf("server not reachable: %w", err)
		}
		if err := runPlaywrightTests(t, baseURL); err != nil {
			return fmt.Errorf("playwright UI smoke: %w", err)
		}
		if err := pollMilestones(t, postUISmokeMilestones, projectDir); err != nil {
			return err
		}
		return nil
	}

	err := runSmokePhases()
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
		configureProcessGroup(cmd)
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

		if retryErr := runSmokePhases(); retryErr != nil {
			t.Fatalf("smoke flow not reached after retry: %v", retryErr)
		}
	}

	// Assertions.
	assertOrdersExist(t, projectDir)
	assertSessionMeta(t, projectDir)
}

func TestSmokeProcessRuntimeDefault(t *testing.T) {
	preflight(t)

	noodleBin := buildNoodle(t)
	dir := newProjectTempDir(t)

	// Scaffold a minimal project with explicit runtime.default.
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
	adapterDir := filepath.Join(dir, "adapters")
	mkdirAll(t, adapterDir)
	writeFile(t, filepath.Join(adapterDir, "backlog-sync"), "#!/bin/sh\necho '{\"id\":\"1\",\"title\":\"hello\",\"status\":\"open\",\"tags\":[]}'\n")
	writeFile(t, filepath.Join(adapterDir, "backlog-done"), "#!/bin/sh\n")
	writeFile(t, filepath.Join(adapterDir, "backlog-add"), "#!/bin/sh\n")
	writeFile(t, filepath.Join(adapterDir, "backlog-edit"), "#!/bin/sh\n")
	chmodExec(t, filepath.Join(adapterDir, "backlog-sync"))
	chmodExec(t, filepath.Join(adapterDir, "backlog-done"))
	chmodExec(t, filepath.Join(adapterDir, "backlog-add"))
	chmodExec(t, filepath.Join(adapterDir, "backlog-edit"))

	// Config with runtime.default = "process".
	writeFile(t, filepath.Join(dir, ".noodle.toml"), `mode = "auto"

[routing.defaults]
provider = "codex"
model = "gpt-5.3-codex-spark"

[skills]
paths = [".agents/skills"]

[agents.codex]
path = "~/.codex"

[adapters.backlog]
skill = "todo"

[adapters.backlog.scripts]
sync = "adapters/backlog-sync"
done = "adapters/backlog-done"
add = "adapters/backlog-add"
edit = "adapters/backlog-edit"

[concurrency]
max_concurrency = 1

[runtime]
default = "process"

[server]
enabled = true
port = 13738
`)

	mkdirAll(t, filepath.Join(dir, ".noodle"))
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "scaffolding")

	// Start noodle.
	cmd := exec.Command(noodleBin, "start")
	configureProcessGroup(cmd)
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

	// Verify snapshot does not report runtime.default as unknown.
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
	for _, w := range snap.Warnings {
		if strings.Contains(w, "unknown runtime") || strings.Contains(w, "runtime.default") {
			t.Fatalf("unexpected runtime.default warning: %v", snap.Warnings)
		}
	}
	t.Logf("snapshot warnings: %v", snap.Warnings)
}

func TestSmokeStartOnceWithoutSkillsInEmptyRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("skipping: git not found on PATH")
	}

	noodleBin := buildNoodle(t)
	dir := newProjectTempDir(t)

	// Empty repo: no project files, no skills, no config.
	run(t, dir, "git", "init", "-b", "main")
	run(t, dir, "git", "config", "user.email", "test@noodle.dev")
	run(t, dir, "git", "config", "user.name", "Noodle Test")
	run(t, dir, "git", "commit", "--allow-empty", "-m", "initial commit")

	skillsDir := filepath.Join(dir, ".agents", "skills")
	if _, err := os.Stat(skillsDir); !os.IsNotExist(err) {
		t.Fatalf("expected no skills directory before startup, got err=%v", err)
	}

	startOnce := func() string {
		t.Helper()
		cmd := exec.Command(noodleBin, "start", "--once")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "NOODLE_NO_BROWSER=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("start --once in empty repo: %v\noutput:\n%s", err, string(out))
		}
		return string(out)
	}

	firstOut := startOnce()
	t.Logf("first start --once output:\n%s", strings.TrimSpace(firstOut))
	secondOut := startOnce()
	t.Logf("second start --once output:\n%s", strings.TrimSpace(secondOut))

	for _, rel := range []string{
		".noodle/status.json",
		".noodle/mise.json",
		".noodle/tickets.json",
		".noodle/control.lock",
		".noodle.toml",
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("expected %s to exist after startup: %v", rel, err)
		}
	}

	statusCmd := exec.Command(noodleBin, "status")
	statusCmd.Dir = dir
	statusCmd.Env = append(os.Environ(), "NOODLE_NO_BROWSER=1")
	statusOut, err := statusCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("noodle status in empty repo: %v\noutput:\n%s", err, string(statusOut))
	}
	t.Logf("status output:\n%s", strings.TrimSpace(string(statusOut)))
	if !strings.Contains(string(statusOut), "loop=") {
		t.Fatalf("status output missing loop state: %s", string(statusOut))
	}
	if !strings.Contains(string(statusOut), "loop=idle") {
		t.Fatalf("status output expected loop=idle in empty repo: %s", string(statusOut))
	}
	if !strings.Contains(string(statusOut), "orders=0") {
		t.Fatalf("status output expected orders=0 in empty repo: %s", string(statusOut))
	}

	if _, err := os.Stat(skillsDir); !os.IsNotExist(err) {
		t.Fatalf("expected no skills directory after startup, got err=%v", err)
	}
}

func TestSmokeStartOnceWorktreeCreateFailureShowsGitError(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("skipping: git not found on PATH")
	}

	noodleBin := buildNoodle(t)
	dir := newProjectTempDir(t)

	run(t, dir, "git", "init", "-b", "main")
	run(t, dir, "git", "config", "user.email", "test@noodle.dev")
	run(t, dir, "git", "config", "user.name", "Noodle Test")
	run(t, dir, "git", "commit", "--allow-empty", "-m", "initial commit")

	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/e2e\n\ngo 1.23\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n")
	writeFile(t, filepath.Join(dir, "brain", "index.md"), "# Brain\n")

	srcSkills := filepath.Join(repoRoot(t), ".agents", "skills")
	dstSkills := filepath.Join(dir, ".agents", "skills")
	copyDir(t, filepath.Join(srcSkills, "execute"), filepath.Join(dstSkills, "execute"))

	mkdirAll(t, filepath.Join(dir, ".noodle"))
	writeFile(t, filepath.Join(dir, ".noodle", "orders.json"), `{
  "orders": [
    {
      "id": "11",
      "title": "Trigger worktree create failure",
      "stages": [
        {
          "task_key": "execute",
          "skill": "execute",
          "provider": "codex",
          "model": "gpt-5.3-codex",
          "status": "pending"
        }
      ],
      "status": "active"
    }
  ]
}`)
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "scaffold")

	branchInUsePath := filepath.Join(newProjectTempDir(t), "branch-in-use")
	run(t, dir, "git", "worktree", "add", "-b", "11-0-execute", branchInUsePath)

	cmd := exec.Command(noodleBin, "start", "--once")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "NOODLE_NO_BROWSER=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected start --once to fail when worktree creation fails\noutput:\n%s", string(out))
	}

	output := string(out)
	t.Logf("start --once output:\n%s", strings.TrimSpace(output))
	if !strings.Contains(output, "cycle spawn cooks failed at cycle.spawn") {
		t.Fatalf("expected cycle.spawn failure in output, got:\n%s", output)
	}
	if !strings.Contains(output, "failed to create worktree") {
		t.Fatalf("expected worktree failure in output, got:\n%s", output)
	}
	if !strings.Contains(output, "branch named '11-0-execute' already exists") {
		t.Fatalf("expected underlying git error details in output, got:\n%s", output)
	}
}

func TestSmokeStartOnceBacklogParseWarningIsRecoverable(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("skipping: git not found on PATH")
	}

	noodleBin := buildNoodle(t)
	dir := newProjectTempDir(t)

	run(t, dir, "git", "init", "-b", "main")
	run(t, dir, "git", "config", "user.email", "test@noodle.dev")
	run(t, dir, "git", "config", "user.name", "Noodle Test")
	run(t, dir, "git", "commit", "--allow-empty", "-m", "initial commit")

	writeFile(t, filepath.Join(dir, "brain", "index.md"), "# Brain\n")
	srcSkills := filepath.Join(repoRoot(t), ".agents", "skills")
	dstSkills := filepath.Join(dir, ".agents", "skills")
	copyDir(t, filepath.Join(srcSkills, "execute"), filepath.Join(dstSkills, "execute"))

	adapterDir := filepath.Join(dir, "adapters")
	mkdirAll(t, adapterDir)
	writeFile(t, filepath.Join(adapterDir, "backlog-sync"), "#!/bin/sh\necho 'P0 malformed backlog line'\n")
	writeFile(t, filepath.Join(adapterDir, "backlog-done"), "#!/bin/sh\n")
	writeFile(t, filepath.Join(adapterDir, "backlog-add"), "#!/bin/sh\n")
	writeFile(t, filepath.Join(adapterDir, "backlog-edit"), "#!/bin/sh\n")
	chmodExec(t, filepath.Join(adapterDir, "backlog-sync"))
	chmodExec(t, filepath.Join(adapterDir, "backlog-done"))
	chmodExec(t, filepath.Join(adapterDir, "backlog-add"))
	chmodExec(t, filepath.Join(adapterDir, "backlog-edit"))

	writeFile(t, filepath.Join(dir, ".noodle.toml"), `mode = "auto"

[routing.defaults]
provider = "codex"
model = "gpt-5.3-codex-spark"

[skills]
paths = [".agents/skills"]

[adapters.backlog]
skill = "todo"

[adapters.backlog.scripts]
sync = "adapters/backlog-sync"
done = "adapters/backlog-done"
add = "adapters/backlog-add"
edit = "adapters/backlog-edit"

[runtime]
default = "process"

[server]
enabled = false
`)

	cmd := exec.Command(noodleBin, "start", "--once")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "NOODLE_NO_BROWSER=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected malformed backlog line to be recoverable\nerr: %v\noutput:\n%s", err, string(out))
	}

	var mise struct {
		Warnings []string `json:"warnings"`
	}
	miseData, readErr := os.ReadFile(filepath.Join(dir, ".noodle", "mise.json"))
	if readErr != nil {
		t.Fatalf("read mise.json: %v", readErr)
	}
	if unmarshalErr := json.Unmarshal(miseData, &mise); unmarshalErr != nil {
		t.Fatalf("parse mise.json: %v", unmarshalErr)
	}
	joinedWarnings := strings.Join(mise.Warnings, "\n")
	if !strings.Contains(joinedWarnings, "parse backlog sync line 1") {
		t.Fatalf("expected parse warning in mise warnings, got: %v", mise.Warnings)
	}
}

// TestSmokeScheduleOnlyNoTaskTypes verifies that noodle starts and the scheduler
// produces orders when no non-schedule task types are registered. With only the
// schedule skill present, the scheduler should create prompt-only (ad-hoc)
// orders for backlog items since there are no task types to bind to.
func TestSmokeScheduleOnlyNoTaskTypes(t *testing.T) {
	preflight(t)

	noodleBin := buildNoodle(t)
	dir := newProjectTempDir(t)

	// Scaffold a minimal project with ONLY the schedule skill.
	run(t, dir, "git", "init", "-b", "main")
	run(t, dir, "git", "config", "user.email", "test@noodle.dev")
	run(t, dir, "git", "config", "user.name", "Noodle Test")
	run(t, dir, "git", "commit", "--allow-empty", "-m", "initial commit")

	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/e2e\n\ngo 1.23\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "init")

	// Brain scaffolding with a backlog item.
	writeFile(t, filepath.Join(dir, "brain", "index.md"), "# Brain\n")
	writeFile(t, filepath.Join(dir, "brain", "todos.md"), "# Todos\n\n<!-- next-id: 2 -->\n\n## Tasks\n\n1. [ ] Create hello.txt ~small\n")

	// Only copy the schedule skill — no execute, quality, or reflect.
	srcSkills := filepath.Join(repoRoot(t), ".agents", "skills")
	dstSkills := filepath.Join(dir, ".agents", "skills")
	copyDir(t, filepath.Join(srcSkills, "schedule"), filepath.Join(dstSkills, "schedule"))

	// Adapter scripts.
	adapterDir := filepath.Join(dir, "adapters")
	mkdirAll(t, adapterDir)
	writeFile(t, filepath.Join(adapterDir, "backlog-sync"), "#!/bin/sh\necho '{\"id\":\"1\",\"title\":\"Create hello.txt\",\"status\":\"open\"}'\n")
	writeFile(t, filepath.Join(adapterDir, "backlog-done"), "#!/bin/sh\n")
	writeFile(t, filepath.Join(adapterDir, "backlog-add"), "#!/bin/sh\n")
	writeFile(t, filepath.Join(adapterDir, "backlog-edit"), "#!/bin/sh\n")
	chmodExec(t, filepath.Join(adapterDir, "backlog-sync"))
	chmodExec(t, filepath.Join(adapterDir, "backlog-done"))
	chmodExec(t, filepath.Join(adapterDir, "backlog-add"))
	chmodExec(t, filepath.Join(adapterDir, "backlog-edit"))

	// Config: schedule-only, codex-spark for speed.
	writeFile(t, filepath.Join(dir, ".noodle.toml"), `mode = "auto"

[routing.defaults]
provider = "codex"
model = "gpt-5.3-codex-spark"

[skills]
paths = [".agents/skills"]

[agents.codex]
path = "~/.codex"

[adapters.backlog]
skill = "todo"

[adapters.backlog.scripts]
sync = "adapters/backlog-sync"
done = "adapters/backlog-done"
add = "adapters/backlog-add"
edit = "adapters/backlog-edit"

[concurrency]
max_concurrency = 1

[runtime]
default = "process"

[server]
enabled = true
port = 13739
`)

	mkdirAll(t, filepath.Join(dir, ".noodle"))
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "scaffolding")

	// Start noodle.
	cmd := exec.Command(noodleBin, "start")
	configureProcessGroup(cmd)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "NOODLE_NO_BROWSER=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	t.Cleanup(func() { cleanupNoodle(t, cmd, dir) })

	if err := cmd.Start(); err != nil {
		t.Fatalf("start noodle: %v", err)
	}
	t.Logf("noodle started (pid %d)", cmd.Process.Pid)

	// The scheduler should produce orders even with no non-schedule task types.
	milestones := []milestone{
		{
			name:    "orders.json appears",
			timeout: 90 * time.Second,
			check:   func(d string) (bool, error) { return ordersExist(d) },
		},
	}
	if err := pollMilestones(t, milestones, dir); err != nil {
		dumpSessionDiagnostics(t, dir)
		t.Fatalf("milestone not reached: %v", err)
	}

	// Read orders and log structure for debugging.
	ordersPath := filepath.Join(dir, ".noodle", "orders.json")
	data, err := os.ReadFile(ordersPath)
	if err != nil {
		t.Fatalf("read orders.json: %v", err)
	}

	var orders struct {
		Orders []struct {
			ID     string `json:"id"`
			Stages []struct {
				TaskKey string `json:"task_key"`
				Prompt  string `json:"prompt"`
			} `json:"stages"`
		} `json:"orders"`
	}
	if err := json.Unmarshal(data, &orders); err != nil {
		t.Fatalf("parse orders.json: %v", err)
	}

	t.Logf("orders.json: %d order(s)", len(orders.Orders))
	for i, order := range orders.Orders {
		for j, stage := range order.Stages {
			t.Logf("  order[%d].stages[%d]: task_key=%q prompt_len=%d", i, j, stage.TaskKey, len(stage.Prompt))
		}
	}

	if len(orders.Orders) == 0 {
		t.Fatalf("expected at least one order, got 0")
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

// TestSmokeInstallMd validates the INSTALL.md onboarding flow end-to-end:
// codex reads INSTALL.md, scaffolds the project, then noodle start --once
// succeeds with a populated mise.json.
func TestSmokeInstallMd(t *testing.T) {
	preflight(t)

	noodleBin := buildNoodle(t)
	dir := newProjectTempDir(t)

	// Bare git repo with dummy Go project files — codex does all noodle scaffolding.
	run(t, dir, "git", "init", "-b", "main")
	run(t, dir, "git", "config", "user.email", "test@noodle.dev")
	run(t, dir, "git", "config", "user.name", "Noodle Test")
	run(t, dir, "git", "commit", "--allow-empty", "-m", "initial commit")

	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/e2e\n\ngo 1.23\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n")
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "init project")

	// Run codex to follow INSTALL.md — the actual onboarding flow.
	root := repoRoot(t)
	installMd := filepath.Join(root, "INSTALL.md")
	prompt := fmt.Sprintf(`Install Noodle and set up this project. Follow %s.

This run is non-interactive. Do not ask follow-up questions.
If INSTALL.md says to ask the user, choose these defaults and continue:
- provider: Codex
- first backlog item in todos.md: "Initial smoke test task"
- brainmaxxing: no

Complete the setup end-to-end and then report completion.`, installMd)

	codexOut := filepath.Join(t.TempDir(), "codex-output.txt")
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	codexCmd := exec.CommandContext(ctx, "codex", "exec",
		"--skip-git-repo-check",
		"--dangerously-bypass-approvals-and-sandbox",
		"--model", "gpt-5.3-codex-spark",
		"-o", codexOut,
		prompt,
	)
	codexCmd.Dir = dir
	codexCmd.Env = append(os.Environ(), "NOODLE_NO_BROWSER=1")

	codexOutput, codexErr := codexCmd.CombinedOutput()
	t.Logf("codex output:\n%s", string(codexOutput))
	if outData, err := os.ReadFile(codexOut); err == nil {
		t.Logf("codex -o file:\n%s", string(outData))
	}
	if codexErr != nil {
		t.Fatalf("codex exec failed: %v", codexErr)
	}

	// Commit everything codex wrote so noodle has clean git state.
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "--allow-empty", "-m", "codex install scaffolding")

	// Run noodle start --once — validates the install produced a working setup.
	startCmd := exec.Command(noodleBin, "start", "--once")
	startCmd.Dir = dir
	startCmd.Env = append(os.Environ(), "NOODLE_NO_BROWSER=1")

	startOut, startErr := startCmd.CombinedOutput()
	t.Logf("noodle start --once output:\n%s", string(startOut))
	if startErr != nil {
		t.Fatalf("noodle start --once failed: %v\noutput:\n%s", startErr, string(startOut))
	}

	// Assertion: .noodle/mise.json exists and parses as valid JSON.
	misePath := filepath.Join(dir, ".noodle", "mise.json")
	miseData, err := os.ReadFile(misePath)
	if err != nil {
		t.Fatalf("mise.json not found: %v", err)
	}

	var mise struct {
		Backlog []struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			Status string `json:"status"`
		} `json:"backlog"`
	}
	if err := json.Unmarshal(miseData, &mise); err != nil {
		t.Fatalf("mise.json invalid JSON: %v\nraw:\n%s", err, string(miseData))
	}

	// Assertion: backlog has at least 1 item.
	if len(mise.Backlog) == 0 {
		t.Fatalf("mise.json backlog is empty — expected at least 1 item\nraw:\n%s", string(miseData))
	}

	// Assertion: at least one backlog item has status "open".
	hasOpen := false
	for _, item := range mise.Backlog {
		t.Logf("backlog item: id=%q title=%q status=%q", item.ID, item.Title, item.Status)
		if item.Status == "open" {
			hasOpen = true
		}
	}
	if !hasOpen {
		t.Fatalf("no backlog item with status=open")
	}
}

// assertSessionMeta verifies at least one non-schedule session reached a
// terminal state according to the same detector used by milestone polling.
func assertSessionMeta(t *testing.T, projectDir string) {
	t.Helper()
	completed, err := sessionCompleted(projectDir)
	if err != nil {
		t.Errorf("check session completion: %v", err)
		return
	}
	if !completed {
		t.Error("no terminal non-schedule session found")
	}
}

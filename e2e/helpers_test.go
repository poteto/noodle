//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// noodleBinary is the path to the compiled noodle binary, built once per test run.
// The binary lives in an os.MkdirTemp directory (not t.TempDir) so it survives
// across tests — t.TempDir is scoped to the creating test and would be cleaned
// up before later tests can use the cached binary.
var (
	noodleBinaryOnce sync.Once
	noodleBinaryPath string
	noodleBinaryErr  error
)

func TestMain(m *testing.M) {
	code := m.Run()
	if noodleBinaryPath != "" {
		os.RemoveAll(filepath.Dir(noodleBinaryPath))
	}
	os.Exit(code)
}

// buildNoodle compiles the noodle binary (and UI assets if missing) and returns
// its path. Uses sync.Once so the build runs at most once per process.
func buildNoodle(t *testing.T) string {
	t.Helper()
	noodleBinaryOnce.Do(func() {
		root := repoRoot(t)

		// Build UI assets if missing so the embedded SPA is functional.
		uiIndex := filepath.Join(root, "ui", "dist", "client", "index.html")
		if _, err := os.Stat(uiIndex); err != nil {
			build := exec.Command("pnpm", "build")
			build.Dir = filepath.Join(root, "ui")
			if out, err := build.CombinedOutput(); err != nil {
				noodleBinaryErr = fmt.Errorf("build ui: %s: %w", string(out), err)
				return
			}
		}

		dir, err := os.MkdirTemp("", "noodle-e2e-*")
		if err != nil {
			noodleBinaryErr = fmt.Errorf("create temp dir: %w", err)
			return
		}
		noodleBinaryPath = filepath.Join(dir, "noodle")
		cmd := exec.Command("go", "build", "-o", noodleBinaryPath, ".")
		cmd.Dir = root
		out, buildErr := cmd.CombinedOutput()
		if buildErr != nil {
			noodleBinaryErr = fmt.Errorf("build noodle: %s: %w", string(out), buildErr)
		}
	})
	if noodleBinaryErr != nil {
		t.Fatalf("build noodle binary: %v", noodleBinaryErr)
	}
	return noodleBinaryPath
}

// repoRoot returns the root of the noodle repository by walking up from the
// test file location.
func repoRoot(t *testing.T) string {
	t.Helper()
	// go test sets the working directory to the package directory (e2e/).
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Dir(wd)
}

func newProjectTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "noodle-e2e-project-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() {
		cleanupTempDir(t, dir)
	})
	return dir
}

func cleanupTempDir(t *testing.T, dir string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		err := os.RemoveAll(dir)
		if err == nil || os.IsNotExist(err) {
			return
		}
		if time.Now().After(deadline) {
			t.Logf("warning: temp dir cleanup incomplete for %s: %v", dir, err)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// scaffoldProject creates a temporary directory with the full project
// structure needed for a noodle E2E run. Returns the project directory path.
func scaffoldProject(t *testing.T, noodleBin string) string {
	t.Helper()

	dir := newProjectTempDir(t)

	// Initialize git repo with initial commit on main.
	run(t, dir, "git", "init", "-b", "main")
	run(t, dir, "git", "config", "user.email", "test@noodle.dev")
	run(t, dir, "git", "config", "user.name", "Noodle Test")
	run(t, dir, "git", "commit", "--allow-empty", "-m", "initial commit")

	// Dummy project files — include go.mod so go vet/test work.
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/e2e\n\ngo 1.23\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import "fmt"

func main() {
	fmt.Println("hello from noodle e2e")
}
`)
	run(t, dir, "git", "add", "go.mod", "main.go")
	run(t, dir, "git", "commit", "-m", "add project files")

	// Brain scaffolding.
	writeFile(t, filepath.Join(dir, "brain", "index.md"), "# Brain\n")

	// Todos with a trivial item.
	writeFile(t, filepath.Join(dir, "todos.md"), `# Todos

<!-- next-id: 2 -->

## Tasks

1. [ ] Create a hello.txt file containing "hello world" ~small
`)

	// Skills directory — copy schedule and execute from the repo.
	srcSkills := filepath.Join(repoRoot(t), ".agents", "skills")
	dstSkills := filepath.Join(dir, ".agents", "skills")
	for _, skill := range []string{"schedule", "execute"} {
		src := filepath.Join(srcSkills, skill)
		dst := filepath.Join(dstSkills, skill)
		copyDir(t, src, dst)
	}

	// Backlog adapter scripts — simple shell scripts for the E2E test.
	adapterDir := filepath.Join(dir, "adapters")
	mkdirAll(t, adapterDir)
	// sync: emit one NDJSON line per open todo.
	writeFile(t, filepath.Join(adapterDir, "backlog-sync"), `#!/bin/sh
set -e
TODOS="todos.md"
[ -f "$TODOS" ] || exit 0
# Match "1. [ ] some title ~small" lines and emit NDJSON.
sed -n 's/^\([0-9]*\)\. \[ \] \(.*\)/\1 \2/p' "$TODOS" | while read -r id rest; do
  title=$(printf '%s' "$rest" | sed 's/ ~[a-z]*$//' | sed 's/"/\\"/g')
  printf '{"id":"%s","title":"%s","status":"open"}\n' "$id" "$title"
done
`)
	writeFile(t, filepath.Join(adapterDir, "backlog-done"), `#!/bin/sh
echo "done: $1" >&2
`)
	writeFile(t, filepath.Join(adapterDir, "backlog-add"), `#!/bin/sh
echo "add: $@" >&2
`)
	writeFile(t, filepath.Join(adapterDir, "backlog-edit"), `#!/bin/sh
echo "edit: $@" >&2
`)
	chmodExec(t, filepath.Join(adapterDir, "backlog-sync"))
	chmodExec(t, filepath.Join(adapterDir, "backlog-done"))
	chmodExec(t, filepath.Join(adapterDir, "backlog-add"))
	chmodExec(t, filepath.Join(adapterDir, "backlog-edit"))

	// .noodle.toml — codex provider, auto mode, max_concurrency=1.
	// Server enabled on a fixed port so Playwright can hit the UI.
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
port = 13737
`)

	// Runtime directory.
	mkdirAll(t, filepath.Join(dir, ".noodle"))

	// Commit scaffolding.
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "noodle e2e scaffolding")

	return dir
}

// milestone represents a phased polling checkpoint.
type milestone struct {
	name    string
	timeout time.Duration
	check   func(projectDir string) (bool, error)
}

// pollMilestones polls milestones sequentially with per-phase deadlines and
// exponential backoff. Returns an error if any milestone is not reached within
// its deadline.
func pollMilestones(t *testing.T, milestones []milestone, projectDir string) error {
	t.Helper()
	for _, ms := range milestones {
		t.Logf("polling milestone: %s (timeout %s)", ms.name, ms.timeout)
		deadline := time.Now().Add(ms.timeout)
		backoff := 500 * time.Millisecond
		maxBackoff := 5 * time.Second

		for {
			ok, err := ms.check(projectDir)
			if err != nil {
				return fmt.Errorf("milestone %s check error: %w", ms.name, err)
			}
			if ok {
				t.Logf("milestone %s reached", ms.name)
				break
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("milestone %s not reached within %s", ms.name, ms.timeout)
			}
			time.Sleep(backoff)
			backoff = backoff * 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
	return nil
}

// ordersExist checks if .noodle/orders.json exists and has at least one order.
func ordersExist(projectDir string) (bool, error) {
	path := filepath.Join(projectDir, ".noodle", "orders.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	var orders struct {
		Orders []json.RawMessage `json:"orders"`
	}
	if err := json.Unmarshal(data, &orders); err != nil {
		return false, nil // treat parse errors as not-ready
	}
	return len(orders.Orders) > 0, nil
}

// sessionDirExists checks if any session directory exists under .noodle/sessions/.
func sessionDirExists(projectDir string) (bool, error) {
	sessionsDir := filepath.Join(projectDir, ".noodle", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return true, nil
		}
	}
	return false, nil
}

// sessionCompleted checks if any non-schedule session reached a terminal state.
// It supports both legacy canonical completion events and current result/meta
// status forms used by newer runtimes.
func sessionCompleted(projectDir string) (bool, error) {
	sessionsDir := filepath.Join(projectDir, ".noodle", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Only check execute (non-schedule) sessions.
		if strings.HasPrefix(entry.Name(), "schedule-") {
			continue
		}
		sessionDir := filepath.Join(sessionsDir, entry.Name())
		canonicalPath := filepath.Join(sessionDir, "canonical.ndjson")
		data, err := os.ReadFile(canonicalPath)
		if err != nil {
			data = nil
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var event struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			}
			if json.Unmarshal([]byte(line), &event) != nil {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(event.Type)) {
			case "complete":
				return true, nil
			case "result":
				msg := strings.ToLower(strings.TrimSpace(event.Message))
				if msg == "turn complete" || msg == "turn failed" || msg == "turn cancelled" || msg == "turn canceled" {
					return true, nil
				}
			}
		}

		metaPath := filepath.Join(sessionDir, "meta.json")
		metaData, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta struct {
			Status string `json:"status"`
		}
		if json.Unmarshal(metaData, &meta) != nil {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(meta.Status)) {
		case "completed", "exited", "failed", "killed", "cancelled", "canceled", "stopped":
			return true, nil
		}
	}
	return false, nil
}

// cleanupNoodle kills a noodle process.
func cleanupNoodle(t *testing.T, cmd *exec.Cmd, _ string) {
	t.Helper()
	if cmd != nil && cmd.Process != nil {
		t.Logf("killing noodle process (pid %d)", cmd.Process.Pid)
		killCommandProcessTree(cmd)
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

// run executes a command in the given directory and fails the test on error.
func run(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run %s %v in %s: %s: %v", name, args, dir, string(out), err)
	}
	return string(out)
}

// writeFile creates parent directories and writes content to a file.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	mkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// mkdirAll creates directories, failing the test on error.
func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

// chmodExec marks a file as executable.
func chmodExec(t *testing.T, path string) {
	t.Helper()
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
}

// runPlaywrightTests installs deps and runs the Playwright UI smoke tests
// against the running noodle server. Returns an error if any test fails.
func runPlaywrightTests(t *testing.T, baseURL string) error {
	t.Helper()
	uiTestDir := filepath.Join(repoRoot(t), "e2e", "ui")

	// Install deps (including playwright browsers).
	install := exec.Command("pnpm", "install")
	install.Dir = uiTestDir
	if out, err := install.CombinedOutput(); err != nil {
		return fmt.Errorf("pnpm install: %s: %w", string(out), err)
	}

	// Ensure chromium is installed for playwright.
	browsers := exec.Command("npx", "playwright", "install", "chromium")
	browsers.Dir = uiTestDir
	if out, err := browsers.CombinedOutput(); err != nil {
		return fmt.Errorf("playwright install chromium: %s: %w", string(out), err)
	}

	// Run the tests.
	cmd := exec.Command("npx", "playwright", "test")
	cmd.Dir = uiTestDir
	cmd.Env = append(os.Environ(), "NOODLE_BASE_URL="+baseURL)
	out, err := cmd.CombinedOutput()
	t.Logf("playwright output:\n%s", string(out))
	if err != nil {
		return fmt.Errorf("playwright tests failed: %w", err)
	}
	return nil
}

// waitForServer polls the server until it responds to /api/snapshot or times out.
func waitForServer(t *testing.T, baseURL string, timeout time.Duration) error {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/api/snapshot")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				t.Logf("server ready at %s", baseURL)
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("server at %s not ready within %s", baseURL, timeout)
}

// copyDir recursively copies src to dst.
func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	mkdirAll(t, dst)
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("read dir %s: %v", src, err)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			copyDir(t, srcPath, dstPath)
			continue
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatalf("read %s: %v", srcPath, err)
		}
		if err := os.WriteFile(dstPath, data, 0o644); err != nil {
			t.Fatalf("write %s: %v", dstPath, err)
		}
	}
}

// dumpSessionDiagnostics logs key files from failed sessions for debugging.
// For NDJSON files, shows event type counts and the last 30 lines (the
// tail reveals whether a completion event was emitted).
func dumpSessionDiagnostics(t *testing.T, projectDir string) {
	t.Helper()
	sessionsDir := filepath.Join(projectDir, ".noodle", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		t.Logf("diagnostics: cannot read sessions dir: %v", err)
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sessionDir := filepath.Join(sessionsDir, entry.Name())

		// Small files: dump fully.
		for _, fname := range []string{"prompt.txt", "stderr.log"} {
			data, err := os.ReadFile(filepath.Join(sessionDir, fname))
			if err != nil {
				continue
			}
			t.Logf("diagnostics [%s/%s]:\n%s", entry.Name(), fname, string(data))
		}

		// NDJSON files: show type counts + tail.
		for _, fname := range []string{"canonical.ndjson", "raw.ndjson"} {
			data, err := os.ReadFile(filepath.Join(sessionDir, fname))
			if err != nil {
				continue
			}
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			typeCounts := map[string]int{}
			for _, line := range lines {
				var obj map[string]interface{}
				if json.Unmarshal([]byte(line), &obj) == nil {
					if typ, ok := obj["type"].(string); ok {
						typeCounts[typ]++
					}
				}
			}
			t.Logf("diagnostics [%s/%s]: %d lines, type counts: %v", entry.Name(), fname, len(lines), typeCounts)

			// Show last 30 lines.
			tail := lines
			if len(tail) > 30 {
				tail = tail[len(tail)-30:]
			}
			t.Logf("diagnostics [%s/%s] TAIL:\n%s", entry.Name(), fname, strings.Join(tail, "\n"))
		}
	}
}

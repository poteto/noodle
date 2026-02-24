package startup_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/startup"
)

// TestCLIIntegrationStartScaffolds builds the noodle binary and verifies
// that `noodle start --once` in a fresh directory scaffolds the project structure.
func TestCLIIntegrationStartScaffolds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build the binary
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "noodle")
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = findProjectRoot(t)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build noodle: %v\n%s", err, out)
	}

	// Create a fake tmux so the binary doesn't fail on tmux check
	fakeTmux := filepath.Join(binDir, "tmux")
	if err := os.WriteFile(fakeTmux, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}

	// Run in a fresh temp directory with a git repo (worktrees need one)
	projectDir := t.TempDir()
	gitInit := exec.Command("git", "init")
	gitInit.Dir = projectDir
	if out, err := gitInit.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	gitCommit := exec.Command("git", "commit", "--allow-empty", "-m", "init")
	gitCommit.Dir = projectDir
	if out, err := gitCommit.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	cmd := exec.Command(binPath, "start", "--once")
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), "PATH="+binDir+":"+os.Getenv("PATH"))

	output, err := cmd.CombinedOutput()
	// start --once exits non-zero in a fresh project because skills haven't
	// been installed yet (that's the agent's job after scaffolding). Verify
	// the failure is from missing skills, not from config or scaffolding.
	if err != nil {
		if !strings.Contains(string(output), "skill") {
			t.Fatalf("noodle start --once failed for unexpected reason:\n%s", output)
		}
	}

	// Verify scaffolded files exist
	for _, path := range []string{
		"brain/index.md",
		"brain/todos.md",
		"brain/principles.md",
		"brain/plans/index.md",
		".noodle",
		".noodle.toml",
	} {
		full := filepath.Join(projectDir, path)
		if _, err := os.Stat(full); err != nil {
			t.Errorf("expected %s to exist after first start: %v", path, err)
		}
	}

	// Verify generated config passes Parse
	configData, err := os.ReadFile(filepath.Join(projectDir, ".noodle.toml"))
	if err != nil {
		t.Fatalf("read scaffolded config: %v", err)
	}
	cfg, err := config.Parse(configData)
	if err != nil {
		t.Fatalf("scaffolded config did not parse: %v", err)
	}
	if cfg.Routing.Defaults.Provider != "claude" {
		t.Errorf("routing.defaults.provider = %q, want claude", cfg.Routing.Defaults.Provider)
	}
	if cfg.Routing.Defaults.Model != "claude-opus-4-6" {
		t.Errorf("routing.defaults.model = %q, want claude-opus-4-6", cfg.Routing.Defaults.Model)
	}

	// Run again — should be idempotent
	cmd2 := exec.Command(binPath, "start", "--once")
	cmd2.Dir = projectDir
	cmd2.Env = append(os.Environ(), "PATH="+binDir+":"+os.Getenv("PATH"))
	_, _ = cmd2.CombinedOutput()

	// Verify config was not overwritten (should still parse identically)
	configData2, err := os.ReadFile(filepath.Join(projectDir, ".noodle.toml"))
	if err != nil {
		t.Fatalf("re-read config: %v", err)
	}
	if string(configData) != string(configData2) {
		t.Fatal("config was modified on second run — not idempotent")
	}
}

// TestCLIIntegrationStartNoTmux verifies that noodle start fails with a
// diagnostic when tmux is missing.
func TestCLIIntegrationStartNoTmux(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build the binary
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "noodle")
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = findProjectRoot(t)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build noodle: %v\n%s", err, out)
	}

	// Run with a PATH that does NOT include tmux
	projectDir := t.TempDir()
	cmd := exec.Command(binPath, "start", "--once")
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), "PATH="+binDir)

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit when tmux is missing")
	}
	if !strings.Contains(string(output), "tmux") {
		t.Fatalf("expected tmux diagnostic in output, got:\n%s", output)
	}
}

// TestScaffoldedConfigValidation verifies the scaffolded config produces
// no fatal diagnostics when tmux is available.
func TestScaffoldedConfigValidation(t *testing.T) {
	dir := t.TempDir()
	var buf strings.Builder
	if err := startup.EnsureProjectStructure(dir, &buf); err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origDir)

	cfg, validation, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	_ = cfg

	// Scaffolded config has no adapters, so no adapter diagnostics.
	// Only possible fatal is tmux missing (environment dependent).
	for _, d := range validation.Diagnostics {
		if d.Severity == config.DiagnosticSeverityFatal && d.Code != config.DiagnosticCodeRuntimeTmuxMissing {
			t.Errorf("unexpected fatal diagnostic: %s: %s", d.FieldPath, d.Message)
		}
	}
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// Walk up from startup/ package to find project root
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root")
		}
		dir = parent
	}
}

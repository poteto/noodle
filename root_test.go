package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommandRegistersAllSubcommands(t *testing.T) {
	root := NewRootCmd()

	expected := []string{
		"start", "skills", "schema", "status", "debug",
		"worktree", "stamp", "dispatch", "mise", "plan",
	}

	found := map[string]bool{}
	for _, cmd := range root.Commands() {
		found[cmd.Name()] = true
	}

	for _, name := range expected {
		if !found[name] {
			t.Fatalf("%q command not registered", name)
		}
	}
}

func TestWorktreeCommandRegistersSubcommands(t *testing.T) {
	root := NewRootCmd()
	var wt *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Name() == "worktree" {
			wt = cmd
			break
		}
	}
	if wt == nil {
		t.Fatal("worktree command not registered")
	}

	expected := map[string]bool{
		"hook": true, "create": true, "merge": true,
		"cleanup": true, "list": true, "prune": true,
	}
	for _, cmd := range wt.Commands() {
		delete(expected, cmd.Name())
	}
	for name := range expected {
		t.Fatalf("worktree %q subcommand not registered", name)
	}
}

func TestPlanCommandRegistersSubcommands(t *testing.T) {
	root := NewRootCmd()
	var pl *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Name() == "plan" {
			pl = cmd
			break
		}
	}
	if pl == nil {
		t.Fatal("plan command not registered")
	}

	expected := map[string]bool{
		"create": true, "done": true, "phase-add": true, "list": true,
	}
	for _, cmd := range pl.Commands() {
		delete(expected, cmd.Name())
	}
	for name := range expected {
		t.Fatalf("plan %q subcommand not registered", name)
	}
}

func TestProjectDirFlagRegistered(t *testing.T) {
	root := NewRootCmd()
	f := root.PersistentFlags().Lookup("project-dir")
	if f == nil {
		t.Fatal("--project-dir flag not registered")
	}
	if f.DefValue != "" {
		t.Fatalf("--project-dir default = %q, want empty", f.DefValue)
	}
}

func TestResolveProjectDirDefaultsToCwd(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	got, err := resolveProjectDir("")
	if err != nil {
		t.Fatal(err)
	}
	if got != cwd {
		t.Fatalf("resolveProjectDir(\"\") = %q, want %q", got, cwd)
	}
}

func TestResolveProjectDirFromFlag(t *testing.T) {
	dir := t.TempDir()
	got, err := resolveProjectDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	// TempDir may return a symlinked path on macOS; compare via EvalSymlinks.
	wantAbs, _ := filepath.EvalSymlinks(dir)
	gotAbs, _ := filepath.EvalSymlinks(got)
	if gotAbs != wantAbs {
		t.Fatalf("resolveProjectDir(%q) = %q, want %q", dir, got, wantAbs)
	}
}

func TestResolveProjectDirFromEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NOODLE_PROJECT_DIR", dir)
	got, err := resolveProjectDir("")
	if err != nil {
		t.Fatal(err)
	}
	wantAbs, _ := filepath.EvalSymlinks(dir)
	gotAbs, _ := filepath.EvalSymlinks(got)
	if gotAbs != wantAbs {
		t.Fatalf("resolveProjectDir(\"\") with env = %q, want %q", got, wantAbs)
	}
}

func TestResolveProjectDirFlagBeatsEnv(t *testing.T) {
	flagDir := t.TempDir()
	envDir := t.TempDir()
	t.Setenv("NOODLE_PROJECT_DIR", envDir)
	got, err := resolveProjectDir(flagDir)
	if err != nil {
		t.Fatal(err)
	}
	wantAbs, _ := filepath.EvalSymlinks(flagDir)
	gotAbs, _ := filepath.EvalSymlinks(got)
	if gotAbs != wantAbs {
		t.Fatalf("flag should beat env: got %q, want %q", got, wantAbs)
	}
}

func TestResolveProjectDirRejectsNonexistent(t *testing.T) {
	_, err := resolveProjectDir("/no/such/path/for/noodle")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestResolveProjectDirRejectsFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "not-a-dir")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	_, err = resolveProjectDir(f.Name())
	if err == nil {
		t.Fatal("expected error for file path")
	}
}

func TestSkillsCommandRegistersSubcommands(t *testing.T) {
	root := NewRootCmd()
	var sk *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Name() == "skills" {
			sk = cmd
			break
		}
	}
	if sk == nil {
		t.Fatal("skills command not registered")
	}

	found := false
	for _, cmd := range sk.Commands() {
		if cmd.Name() == "list" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("skills list subcommand not registered")
	}
}

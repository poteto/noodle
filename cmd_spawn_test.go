package main

import (
	"context"
	"strings"
	"testing"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/spawner"
)

func TestRunSpawnCommandRequiresPrompt(t *testing.T) {
	err := runSpawnCommand(context.Background(), nil, nil, []string{"--worktree", ".worktrees/phase-06-spawner"})
	if err == nil {
		t.Fatal("expected prompt required error")
	}
	if !strings.Contains(err.Error(), "prompt is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunSpawnCommandRequiresWorktree(t *testing.T) {
	err := runSpawnCommand(context.Background(), nil, nil, []string{"--prompt", "Say ok"})
	if err == nil {
		t.Fatal("expected worktree required error")
	}
	if !strings.Contains(err.Error(), "worktree is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunSpawnCommandBuildsRequestFromFlagsAndDefaults(t *testing.T) {
	originalFactory := newSpawnCommandSpawner
	t.Cleanup(func() { newSpawnCommandSpawner = originalFactory })

	fake := &fakeSpawnCommandSpawner{
		session: fakeSession{id: "session-123"},
	}
	newSpawnCommandSpawner = func(_ spawner.TmuxSpawnerConfig) spawnCommandSpawner {
		return fake
	}

	app := &App{
		Config:     config.DefaultConfig(),
		Validation: config.ValidationResult{},
	}
	app.Config.Routing.Defaults.Provider = "codex"
	app.Config.Routing.Defaults.Model = "gpt-5.3-codex"
	app.Config.Skills.Paths = []string{"skills", "~/.noodle/skills"}
	app.Config.Agents.ClaudeDir = "/tmp/claude-bin"
	app.Config.Agents.CodexDir = "/tmp/codex-bin"

	var err error
	_ = captureStdout(t, func() {
		err = runSpawnCommand(context.Background(), app, nil, []string{
			"--name", "cook-a",
			"--prompt", "Say ok",
			"--skill", "debugging",
			"--reasoning-level", "high",
			"--worktree", ".worktrees/phase-06-spawner",
			"--max-turns", "7",
			"--budget-cap", "1.25",
			"--env", "FOO=bar",
			"--env", "BAR=baz",
		})
	})
	if err != nil {
		t.Fatalf("runSpawnCommand returned error: %v", err)
	}
	if fake.called != 1 {
		t.Fatalf("spawn called %d times, expected 1", fake.called)
	}
	if fake.req.Name != "cook-a" {
		t.Fatalf("request name = %q", fake.req.Name)
	}
	if fake.req.Prompt != "Say ok" {
		t.Fatalf("request prompt = %q", fake.req.Prompt)
	}
	if fake.req.Provider != "codex" {
		t.Fatalf("request provider = %q", fake.req.Provider)
	}
	if fake.req.Model != "gpt-5.3-codex" {
		t.Fatalf("request model = %q", fake.req.Model)
	}
	if fake.req.Skill != "debugging" {
		t.Fatalf("request skill = %q", fake.req.Skill)
	}
	if fake.req.ReasoningLevel != "high" {
		t.Fatalf("request reasoning level = %q", fake.req.ReasoningLevel)
	}
	if fake.req.WorktreePath != ".worktrees/phase-06-spawner" {
		t.Fatalf("request worktree path = %q", fake.req.WorktreePath)
	}
	if fake.req.MaxTurns != 7 {
		t.Fatalf("request max turns = %d", fake.req.MaxTurns)
	}
	if fake.req.BudgetCap != 1.25 {
		t.Fatalf("request budget cap = %f", fake.req.BudgetCap)
	}
	if fake.req.EnvVars["FOO"] != "bar" || fake.req.EnvVars["BAR"] != "baz" {
		t.Fatalf("request env vars = %#v", fake.req.EnvVars)
	}
}

type fakeSpawnCommandSpawner struct {
	req     spawner.SpawnRequest
	session spawner.Session
	err     error
	called  int
}

func (f *fakeSpawnCommandSpawner) Spawn(_ context.Context, req spawner.SpawnRequest) (spawner.Session, error) {
	f.called++
	f.req = req
	if f.err != nil {
		return nil, f.err
	}
	return f.session, nil
}

type fakeSession struct {
	id string
}

func (s fakeSession) ID() string                          { return s.id }
func (s fakeSession) Status() string                      { return "running" }
func (s fakeSession) Events() <-chan spawner.SessionEvent { return make(chan spawner.SessionEvent) }
func (s fakeSession) Done() <-chan struct{}               { return make(chan struct{}) }
func (s fakeSession) TotalCost() float64                  { return 0 }
func (s fakeSession) Kill() error                         { return nil }

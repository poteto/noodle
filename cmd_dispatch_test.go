package main

import (
	"context"
	"strings"
	"testing"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/dispatcher"
)

func TestRunDispatchRequiresPrompt(t *testing.T) {
	err := runDispatch(context.Background(), nil, dispatchArgs{worktree: ".worktrees/phase-06-dispatcher"})
	if err == nil {
		t.Fatal("expected prompt required error")
	}
	if !strings.Contains(err.Error(), "prompt is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDispatchRequiresWorktree(t *testing.T) {
	err := runDispatch(context.Background(), nil, dispatchArgs{prompt: "Say ok"})
	if err == nil {
		t.Fatal("expected worktree required error")
	}
	if !strings.Contains(err.Error(), "worktree is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDispatchBuildsRequestFromDefaults(t *testing.T) {
	originalFactory := newDispatchCommandDispatcher
	t.Cleanup(func() { newDispatchCommandDispatcher = originalFactory })

	fake := &fakeDispatchCommandDispatcher{
		session: fakeSession{id: "session-123"},
	}
	newDispatchCommandDispatcher = func(_ dispatcher.TmuxDispatcherConfig) dispatchCommandDispatcher {
		return fake
	}

	app := &App{
		Config:     config.DefaultConfig(),
		Validation: config.ValidationResult{},
	}
	app.Config.Routing.Defaults.Provider = "codex"
	app.Config.Routing.Defaults.Model = "gpt-5.3-codex"
	app.Config.Skills.Paths = []string{"skills", "~/.noodle/skills"}
	app.Config.Agents.Claude.Path = "/tmp/claude-bin"
	app.Config.Agents.Codex.Path = "/tmp/codex-bin"

	var err error
	_ = captureStdout(t, func() {
		err = runDispatch(context.Background(), app, dispatchArgs{
			name:           "cook-a",
			prompt:         "Say ok",
			skill:          "debugging",
			reasoningLevel: "high",
			worktree:       ".worktrees/phase-06-dispatcher",
			maxTurns:       7,
			budgetCap:      1.25,
			envVars:        map[string]string{"FOO": "bar", "BAR": "baz"},
		})
	})
	if err != nil {
		t.Fatalf("runDispatch returned error: %v", err)
	}
	if fake.called != 1 {
		t.Fatalf("dispatch called %d times, expected 1", fake.called)
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
	if fake.req.WorktreePath != ".worktrees/phase-06-dispatcher" {
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

type fakeDispatchCommandDispatcher struct {
	req     dispatcher.DispatchRequest
	session dispatcher.Session
	err     error
	called  int
}

func (f *fakeDispatchCommandDispatcher) Dispatch(_ context.Context, req dispatcher.DispatchRequest) (dispatcher.Session, error) {
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

func (s fakeSession) ID() string                              { return s.id }
func (s fakeSession) Status() string                          { return "running" }
func (s fakeSession) Events() <-chan dispatcher.SessionEvent { return make(chan dispatcher.SessionEvent) }
func (s fakeSession) Done() <-chan struct{}                   { return make(chan struct{}) }
func (s fakeSession) TotalCost() float64                      { return 0 }
func (s fakeSession) Kill() error                              { return nil }
func (s fakeSession) Controller() dispatcher.AgentController   { return dispatcher.NoopController() }

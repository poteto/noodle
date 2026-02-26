package loop

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/internal/taskreg"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/skill"
)

// testRegistryWithoutSchedule returns a registry that has execute but no
// schedule skill — simulating the "missing schedule" scenario.
func testRegistryWithoutSchedule() taskreg.Registry {
	return taskreg.NewFromSkills([]skill.SkillMeta{
		{
			Name: "execute",
			Path: "/skills/execute",
			Frontmatter: skill.Frontmatter{
				Noodle: &skill.NoodleMeta{Schedule: "When a planned item is ready"},
			},
		},
	})
}

// bootstrapMise returns a mise brief with a plan so the loop bootstraps
// a schedule queue instead of going idle.
func bootstrapMise() *fakeMise {
	return &fakeMise{
		brief: mise.Brief{
			Plans: []mise.PlanSummary{{ID: 1, Title: "test plan", Status: "active"}},
		},
	}
}

// createBootstrapSkillFixture creates a bootstrap SKILL.md in a temp
// skill search path and returns the search path for config.
func createBootstrapSkillFixture(t *testing.T, dir string) string {
	t.Helper()
	skillDir := filepath.Join(dir, ".agents", "skills", "bootstrap")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("create bootstrap skill dir: %v", err)
	}
	content := `---
name: bootstrap
description: Creates a schedule skill for new projects.
---

# Bootstrap: Create Schedule Skill

You are a bootstrap agent. Read history from {{history_dirs}}.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write bootstrap SKILL.md: %v", err)
	}
	return filepath.Join(dir, ".agents", "skills")
}

// bootstrapConfig returns a config with skill search paths pointing to the fixture.
func bootstrapConfig(skillSearchPath string) config.Config {
	cfg := config.DefaultConfig()
	cfg.Skills.Paths = []string{skillSearchPath}
	return cfg
}

func TestMissingScheduleSkillTriggersBootstrap(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	skillPath := createBootstrapSkillFixture(t, projectDir)

	sp := &fakeDispatcher{}
	l := New(projectDir, "noodle", bootstrapConfig(skillPath), Dependencies{
		Dispatcher: sp,
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       bootstrapMise(),
		Monitor:    fakeMonitor{},
		Registry:   testRegistryWithoutSchedule(),
		Now:        time.Now,
		QueueFile:  queuePath,
	})

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	if len(sp.calls) != 1 {
		t.Fatalf("expected 1 dispatch call (bootstrap), got %d", len(sp.calls))
	}
	req := sp.calls[0]

	// Bootstrap session name has bootstrap- prefix.
	if !strings.HasPrefix(req.Name, bootstrapSessionPrefix) {
		t.Fatalf("expected bootstrap prefix on name %q", req.Name)
	}

	// Bootstrap uses SystemPrompt, not Skill.
	if strings.TrimSpace(req.SystemPrompt) == "" {
		t.Fatal("expected SystemPrompt to be set for bootstrap dispatch")
	}
	if strings.TrimSpace(req.Skill) != "" {
		t.Fatalf("expected Skill to be empty for bootstrap, got %q", req.Skill)
	}

	// Bootstrap session is tracked in bootstrapInFlight, not activeByID.
	if l.bootstrapInFlight == nil {
		t.Fatal("expected bootstrapInFlight to be set")
	}
	if len(l.activeByID) != 0 {
		t.Fatalf("bootstrap should not be in activeByID, got %d entries", len(l.activeByID))
	}
}

func TestBootstrapSessionUsesSystemPromptNotSkill(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	skillPath := createBootstrapSkillFixture(t, projectDir)

	sp := &fakeDispatcher{}
	l := New(projectDir, "noodle", bootstrapConfig(skillPath), Dependencies{
		Dispatcher: sp,
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       bootstrapMise(),
		Monitor:    fakeMonitor{},
		Registry:   testRegistryWithoutSchedule(),
		Now:        time.Now,
		QueueFile:  queuePath,
	})

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	if len(sp.calls) != 1 {
		t.Fatalf("expected 1 dispatch, got %d", len(sp.calls))
	}
	req := sp.calls[0]

	if req.SystemPrompt == "" {
		t.Fatal("SystemPrompt not set on bootstrap dispatch request")
	}
	if req.Skill != "" {
		t.Fatalf("Skill should be empty for bootstrap, got %q", req.Skill)
	}
	if !strings.Contains(req.SystemPrompt, "Create Schedule Skill") {
		t.Fatal("SystemPrompt does not contain expected bootstrap instructions")
	}
}

func TestFailedBootstrapIncrements(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	skillPath := createBootstrapSkillFixture(t, projectDir)

	sp := &fakeDispatcher{}
	l := New(projectDir, "noodle", bootstrapConfig(skillPath), Dependencies{
		Dispatcher: sp,
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       bootstrapMise(),
		Monitor:    fakeMonitor{},
		Registry:   testRegistryWithoutSchedule(),
		Now:        time.Now,
		QueueFile:  queuePath,
	})

	// First cycle: bootstrap dispatched.
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if l.bootstrapInFlight == nil {
		t.Fatal("expected bootstrap in flight after cycle 1")
	}

	// Simulate bootstrap failure.
	session := sp.sessions[0]
	session.status = "failed"
	close(session.done)

	// Second cycle: collects failed bootstrap (attempt 1), then the cycle
	// continues and re-dispatches a new bootstrap (attempt < 3).
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	if l.bootstrapAttempts != 1 {
		t.Fatalf("bootstrapAttempts = %d, want 1", l.bootstrapAttempts)
	}
	if l.bootstrapExhausted {
		t.Fatal("should not be exhausted after 1 failure")
	}
	// A new bootstrap was dispatched in the same cycle.
	if l.bootstrapInFlight == nil {
		t.Fatal("expected new bootstrap to be dispatched after failure")
	}
}

func TestBootstrapExhaustedAfterThreeFailures(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	skillPath := createBootstrapSkillFixture(t, projectDir)

	sp := &fakeDispatcher{}
	l := New(projectDir, "noodle", bootstrapConfig(skillPath), Dependencies{
		Dispatcher: sp,
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       bootstrapMise(),
		Monitor:    fakeMonitor{},
		Registry:   testRegistryWithoutSchedule(),
		Now:        time.Now,
		QueueFile:  queuePath,
	})

	for i := 0; i < 3; i++ {
		// Cycle to dispatch bootstrap.
		if err := l.Cycle(context.Background()); err != nil {
			t.Fatalf("dispatch cycle %d: %v", i, err)
		}
		if l.bootstrapInFlight == nil {
			if l.bootstrapExhausted {
				break
			}
			t.Fatalf("expected bootstrap in flight at attempt %d", i)
		}

		// Simulate failure.
		session := sp.sessions[len(sp.sessions)-1]
		session.status = "failed"
		close(session.done)

		// Cycle to collect failure.
		if err := l.Cycle(context.Background()); err != nil {
			t.Fatalf("collect cycle %d: %v", i, err)
		}
	}

	if !l.bootstrapExhausted {
		t.Fatal("expected bootstrapExhausted to be true after 3 failures")
	}
	if l.bootstrapAttempts != 3 {
		t.Fatalf("bootstrapAttempts = %d, want 3", l.bootstrapAttempts)
	}
}

func TestBootstrapExhaustionEmitsFeedEvent(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	skillPath := createBootstrapSkillFixture(t, projectDir)

	sp := &fakeDispatcher{}
	l := New(projectDir, "noodle", bootstrapConfig(skillPath), Dependencies{
		Dispatcher: sp,
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       bootstrapMise(),
		Monitor:    fakeMonitor{},
		Registry:   testRegistryWithoutSchedule(),
		Now:        time.Now,
		QueueFile:  queuePath,
	})

	// Exhaust bootstrap attempts.
	for i := 0; i < 3; i++ {
		if err := l.Cycle(context.Background()); err != nil {
			t.Fatalf("dispatch cycle %d: %v", i, err)
		}
		if l.bootstrapInFlight == nil {
			if l.bootstrapExhausted {
				break
			}
			t.Fatalf("expected bootstrap in flight at attempt %d", i)
		}
		session := sp.sessions[len(sp.sessions)-1]
		session.status = "failed"
		close(session.done)
		if err := l.Cycle(context.Background()); err != nil {
			t.Fatalf("collect cycle %d: %v", i, err)
		}
	}

	if !l.bootstrapExhausted {
		t.Fatal("expected bootstrap to be exhausted")
	}

	// One more cycle triggers the exhaustion warning + feed event.
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("post-exhaustion cycle: %v", err)
	}

	eventsPath := filepath.Join(runtimeDir, "queue-events.ndjson")
	data, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if !strings.Contains(string(data), "bootstrap_exhausted") {
		t.Fatal("expected bootstrap_exhausted event in queue-events.ndjson")
	}
	if !strings.Contains(string(data), "create .agents/skills/schedule/SKILL.md manually") {
		t.Fatal("expected actionable message in bootstrap_exhausted event")
	}
}

func TestSuccessfulBootstrapTriggersRebuild(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	skillPath := createBootstrapSkillFixture(t, projectDir)

	sp := &fakeDispatcher{}
	l := New(projectDir, "noodle", bootstrapConfig(skillPath), Dependencies{
		Dispatcher: sp,
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       bootstrapMise(),
		Monitor:    fakeMonitor{},
		Registry:   testRegistryWithoutSchedule(),
		Now:        time.Now,
		QueueFile:  queuePath,
	})

	// Dispatch bootstrap.
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if l.bootstrapInFlight == nil {
		t.Fatal("expected bootstrap in flight")
	}

	// Simulate successful completion.
	session := sp.sessions[0]
	session.status = "completed"
	close(session.done)

	// Collect completion — the cycle will also try to re-dispatch schedule
	// (still missing on disk) which triggers another bootstrap.
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}

	// Success should not increment the failure counter.
	if l.bootstrapAttempts != 0 {
		t.Fatalf("bootstrapAttempts = %d, want 0 (success should not increment)", l.bootstrapAttempts)
	}
	if l.bootstrapExhausted {
		t.Fatal("should not be exhausted after successful bootstrap")
	}

	// Verify bootstrap_complete event was written.
	eventsPath := filepath.Join(runtimeDir, "queue-events.ndjson")
	data, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if !strings.Contains(string(data), "bootstrap_complete") {
		t.Fatal("expected bootstrap_complete event in queue-events.ndjson")
	}
}

func TestLoopContinuesWithExhaustedBootstrap(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	skillPath := createBootstrapSkillFixture(t, projectDir)
	// Also install execute skill on disk so registry rebuild finds it.
	executeSkillDir := filepath.Join(skillPath, "execute")
	if err := os.MkdirAll(executeSkillDir, 0o755); err != nil {
		t.Fatalf("mkdir execute skill: %v", err)
	}
	executeContent := "---\nname: execute\ndescription: Execute tasks\nnoodle:\n  schedule: \"When ready\"\n---\n# Execute\n"
	if err := os.WriteFile(filepath.Join(executeSkillDir, "SKILL.md"), []byte(executeContent), 0o644); err != nil {
		t.Fatalf("write execute SKILL.md: %v", err)
	}
	// Queue has both schedule and a normal item.
	queue := Queue{Items: []QueueItem{
		{ID: "schedule", TaskKey: "schedule", Provider: "claude", Model: "claude-opus-4-6", Skill: "schedule"},
		{ID: "42", Title: "fix bug", Provider: "claude", Model: "claude-opus-4-6", Skill: "execute", TaskKey: "execute"},
	}}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}
	if err := writeOrdersAtomic(ordersPath, queueToOrders(queue)); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	sp := &fakeDispatcher{}
	l := New(projectDir, "noodle", bootstrapConfig(skillPath), Dependencies{
		Dispatcher: sp,
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       bootstrapMise(),
		Monitor:    fakeMonitor{},
		Registry:   testRegistryWithoutSchedule(),
		Now:        time.Now,
		QueueFile:  queuePath,
		OrdersFile: ordersPath,
	})
	l.bootstrapExhausted = true

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	// With bootstrap exhausted and schedule skill missing, the schedule
	// item should be silently skipped. The normal execute item should still
	// be dispatched (if execute skill exists in registry).
	normalCalls := 0
	for _, call := range sp.calls {
		if !strings.HasPrefix(call.Name, bootstrapSessionPrefix) {
			normalCalls++
		}
	}
	if normalCalls != 1 {
		t.Fatalf("expected 1 normal dispatch (execute item), got %d normal out of %d total", normalCalls, len(sp.calls))
	}
}

func TestBootstrapPromptContainsHistoryDirForProvider(t *testing.T) {
	cases := []struct {
		provider string
		expected string
	}{
		{"claude", ".claude/"},
		{"codex", ".codex/"},
		{"other", ".claude/"},
	}
	for _, tc := range cases {
		prompt := buildBootstrapPrompt(tc.provider)
		if !strings.Contains(prompt, tc.expected) {
			t.Fatalf("provider=%q: prompt missing %q", tc.provider, tc.expected)
		}
	}
}

func TestBootstrapPromptIsNonEmpty(t *testing.T) {
	prompt := buildBootstrapPrompt("claude")
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !strings.Contains(prompt, "bootstrap") {
		t.Fatal("expected prompt to contain 'bootstrap'")
	}
}

func TestIsBootstrapSession(t *testing.T) {
	if !isBootstrapSession("bootstrap-schedule") {
		t.Fatal("expected bootstrap-schedule to be bootstrap session")
	}
	if isBootstrapSession("schedule") {
		t.Fatal("expected schedule to NOT be bootstrap session")
	}
	if isBootstrapSession("cook-42") {
		t.Fatal("expected cook-42 to NOT be bootstrap session")
	}
}

func TestSystemPromptFieldOnDispatchRequest(t *testing.T) {
	// Verify that SystemPrompt field is accepted by DispatchRequest validation.
	req := dispatcher.DispatchRequest{
		Name:         "bootstrap-test",
		Prompt:       "test prompt",
		Provider:     "claude",
		Model:        "claude-opus-4-6",
		WorktreePath: "/tmp/test",
		SystemPrompt: "You are a bootstrap agent.",
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

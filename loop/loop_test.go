package loop

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/monitor"
	"github.com/poteto/noodle/spawner"
)

type fakeSession struct {
	id     string
	status string
	done   chan struct{}
}

func (s *fakeSession) ID() string                          { return s.id }
func (s *fakeSession) Status() string                      { return s.status }
func (s *fakeSession) Events() <-chan spawner.SessionEvent { return make(chan spawner.SessionEvent) }
func (s *fakeSession) Done() <-chan struct{}               { return s.done }
func (s *fakeSession) TotalCost() float64                  { return 0 }
func (s *fakeSession) Kill() error                         { s.status = "killed"; close(s.done); return nil }

type fakeSpawner struct {
	calls    []spawner.SpawnRequest
	sessions []*fakeSession
	spawnErr error
}

func (f *fakeSpawner) Spawn(_ context.Context, req spawner.SpawnRequest) (spawner.Session, error) {
	f.calls = append(f.calls, req)
	if f.spawnErr != nil {
		return nil, f.spawnErr
	}
	s := &fakeSession{id: req.Name + "-id", status: "running", done: make(chan struct{})}
	f.sessions = append(f.sessions, s)
	return s, nil
}

type fakeWorktree struct {
	created         []string
	merged          []string
	cleaned         []string
	createErr       error
	createErrByName map[string]error
}

func (f *fakeWorktree) Create(name string) error {
	f.created = append(f.created, name)
	if f.createErrByName != nil {
		if err, ok := f.createErrByName[name]; ok {
			return err
		}
	}
	return f.createErr
}
func (f *fakeWorktree) Merge(name string) error {
	f.merged = append(f.merged, name)
	return nil
}
func (f *fakeWorktree) Cleanup(name string, _ bool) error {
	f.cleaned = append(f.cleaned, name)
	return nil
}

type fakeAdapterRunner struct {
	doneCalls []string
}

func (f *fakeAdapterRunner) Run(_ context.Context, adapterName, action string, opts adapter.RunOptions) (string, error) {
	if adapterName == "backlog" && action == "done" && len(opts.Args) > 0 {
		f.doneCalls = append(f.doneCalls, opts.Args[0])
	}
	return "", nil
}

type fakeMise struct {
	brief    mise.Brief
	warnings []string
	err      error
	results  []fakeMiseResult
	calls    int
}

func (f *fakeMise) Build(_ context.Context) (mise.Brief, []string, error) {
	f.calls++
	if len(f.results) > 0 {
		index := f.calls - 1
		if index >= len(f.results) {
			index = len(f.results) - 1
		}
		current := f.results[index]
		return current.brief, current.warnings, current.err
	}
	return f.brief, f.warnings, f.err
}

type fakeMiseResult struct {
	brief    mise.Brief
	warnings []string
	err      error
}

type fakeMonitor struct{}

func (fakeMonitor) RunOnce(_ context.Context) ([]monitor.SessionMeta, error) {
	return nil, nil
}

func TestCycleSpawnsCookFromQueue(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	review := false
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-sonnet-4-6", Review: &review}}}
	if err := writeQueueAtomic(filepath.Join(runtimeDir, "queue.json"), queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	sp := &fakeSpawner{}
	wt := &fakeWorktree{}
	ar := &fakeAdapterRunner{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Spawner:   sp,
		Worktree:  wt,
		Adapter:   ar,
		Mise:      &fakeMise{},
		Monitor:   fakeMonitor{},
		Registry:  testLoopRegistry(),
		Now:       time.Now,
		QueueFile: filepath.Join(runtimeDir, "queue.json"),
	})
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	if len(sp.calls) != 1 {
		t.Fatalf("spawn calls = %d", len(sp.calls))
	}
	if len(wt.created) != 1 {
		t.Fatalf("worktree creates = %d", len(wt.created))
	}
}

func TestCycleReusesExistingWorktree(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	review := false
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-sonnet-4-6", Review: &review}}}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	existingWorktree := filepath.Join(projectDir, ".worktrees", "42")
	if err := os.MkdirAll(existingWorktree, 0o755); err != nil {
		t.Fatalf("mkdir existing worktree: %v", err)
	}

	sp := &fakeSpawner{}
	wt := &fakeWorktree{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Spawner:   sp,
		Worktree:  wt,
		Adapter:   &fakeAdapterRunner{},
		Mise:      &fakeMise{},
		Monitor:   fakeMonitor{},
		Registry:  testLoopRegistry(),
		Now:       time.Now,
		QueueFile: queuePath,
	})
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	if len(wt.created) != 0 {
		t.Fatalf("expected no worktree create calls, got %d", len(wt.created))
	}
	if len(sp.calls) != 1 {
		t.Fatalf("spawn calls = %d", len(sp.calls))
	}
	if sp.calls[0].WorktreePath != existingWorktree {
		t.Fatalf("spawn worktree path = %q, want %q", sp.calls[0].WorktreePath, existingWorktree)
	}
}

func TestCycleIgnoresDuplicateWorktreeCreateError(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	review := false
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-sonnet-4-6", Review: &review}}}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	existingWorktree := filepath.Join(projectDir, ".worktrees", "42")
	sp := &fakeSpawner{}
	wt := &fakeWorktree{createErr: errors.New("worktree '42' already exists at " + existingWorktree)}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Spawner:   sp,
		Worktree:  wt,
		Adapter:   &fakeAdapterRunner{},
		Mise:      &fakeMise{},
		Monitor:   fakeMonitor{},
		Registry:  testLoopRegistry(),
		Now:       time.Now,
		QueueFile: queuePath,
	})
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	if len(wt.created) != 1 {
		t.Fatalf("expected one create call, got %d", len(wt.created))
	}
	if len(sp.calls) != 1 {
		t.Fatalf("spawn calls = %d", len(sp.calls))
	}
}

func TestCycleSpawnFailureDoesNotCleanupReusedWorktree(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	review := false
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-sonnet-4-6", Review: &review}}}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(projectDir, ".worktrees", "42"), 0o755); err != nil {
		t.Fatalf("mkdir existing worktree: %v", err)
	}

	sp := &fakeSpawner{spawnErr: errors.New("spawn failed")}
	wt := &fakeWorktree{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Spawner:   sp,
		Worktree:  wt,
		Adapter:   &fakeAdapterRunner{},
		Mise:      &fakeMise{},
		Monitor:   fakeMonitor{},
		Registry:  testLoopRegistry(),
		Now:       time.Now,
		QueueFile: queuePath,
	})

	err := l.Cycle(context.Background())
	if err == nil || !strings.Contains(err.Error(), "runtime repair unavailable") {
		t.Fatalf("expected runtime repair error, got %v", err)
	}
	for _, name := range wt.cleaned {
		if name == "42" {
			t.Fatalf("expected no cleanup for reused worktree, got %#v", wt.cleaned)
		}
	}
}

func TestCycleSpawnFailureCleansUpNewWorktree(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	review := false
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-sonnet-4-6", Review: &review}}}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	sp := &fakeSpawner{spawnErr: errors.New("spawn failed")}
	wt := &fakeWorktree{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Spawner:   sp,
		Worktree:  wt,
		Adapter:   &fakeAdapterRunner{},
		Mise:      &fakeMise{},
		Monitor:   fakeMonitor{},
		Registry:  testLoopRegistry(),
		Now:       time.Now,
		QueueFile: queuePath,
	})

	err := l.Cycle(context.Background())
	if err == nil || !strings.Contains(err.Error(), "runtime repair unavailable") {
		t.Fatalf("expected runtime repair error, got %v", err)
	}
	created42 := false
	for _, name := range wt.created {
		if name == "42" {
			created42 = true
			break
		}
	}
	if !created42 {
		t.Fatalf("expected create call for new worktree, got %#v", wt.created)
	}
	cleaned42 := false
	for _, name := range wt.cleaned {
		if name == "42" {
			cleaned42 = true
			break
		}
	}
	if !cleaned42 {
		t.Fatalf("expected cleanup for newly created worktree, got %#v", wt.cleaned)
	}
}

func TestCycleCompletesCookAndMarksDone(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	review := false
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-sonnet-4-6", Review: &review}}}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	sp := &fakeSpawner{}
	wt := &fakeWorktree{}
	ar := &fakeAdapterRunner{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Spawner:   sp,
		Worktree:  wt,
		Adapter:   ar,
		Mise:      &fakeMise{},
		Monitor:   fakeMonitor{},
		Registry:  testLoopRegistry(),
		Now:       time.Now,
		QueueFile: queuePath,
	})
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("spawn cycle: %v", err)
	}
	if len(sp.sessions) != 1 {
		t.Fatalf("sessions = %d", len(sp.sessions))
	}
	sp.sessions[0].status = "completed"
	close(sp.sessions[0].done)

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("completion cycle: %v", err)
	}
	if len(wt.merged) != 1 {
		t.Fatalf("worktree merges = %d", len(wt.merged))
	}
	if len(ar.doneCalls) != 1 || ar.doneCalls[0] != "42" {
		t.Fatalf("backlog done calls = %#v", ar.doneCalls)
	}
	updated, err := readQueue(queuePath)
	if err != nil {
		t.Fatalf("read queue after completion: %v", err)
	}
	if len(updated.Items) != 1 || updated.Items[0].ID != PrioritizeTaskKey() {
		t.Fatalf("expected prioritize bootstrap item after completion, got %#v", updated.Items)
	}
}

func TestCycleBootstrapsPrioritizeUsesRegistrySkill(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")

	sp := &fakeSpawner{}
	wt := &fakeWorktree{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Spawner:   sp,
		Worktree:  wt,
		Adapter:   &fakeAdapterRunner{},
		Mise:      &fakeMise{},
		Monitor:   fakeMonitor{},
		Registry:  testLoopRegistry(),
		Now:       time.Now,
		QueueFile: queuePath,
	})
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	if len(sp.calls) != 1 {
		t.Fatalf("spawn calls = %d", len(sp.calls))
	}
	if sp.calls[0].Skill != "prioritize" {
		t.Fatalf("spawn skill = %q", sp.calls[0].Skill)
	}
	if !strings.Contains(sp.calls[0].Prompt, "Use Skill(prioritize) to refresh .noodle/queue.json from .noodle/mise.json.") {
		t.Fatalf("spawn prompt missing skill invocation: %q", sp.calls[0].Prompt)
	}
	if !strings.Contains(sp.calls[0].Prompt, "queue.json schema (JSON):") {
		t.Fatalf("spawn prompt missing queue schema: %q", sp.calls[0].Prompt)
	}
	if !strings.Contains(sp.calls[0].Prompt, "Task types you may schedule:") {
		t.Fatalf("spawn prompt missing task type catalog: %q", sp.calls[0].Prompt)
	}
	if !strings.Contains(sp.calls[0].Prompt, "- prioritize: ") || !strings.Contains(sp.calls[0].Prompt, "- execute: ") {
		t.Fatalf("spawn prompt missing key+schedule task type guidance: %q", sp.calls[0].Prompt)
	}
	if strings.Contains(sp.calls[0].Prompt, "| config: ") || strings.Contains(sp.calls[0].Prompt, "| synthetic: ") {
		t.Fatalf("spawn prompt should not include verbose task type metadata: %q", sp.calls[0].Prompt)
	}
	if !strings.Contains(sp.calls[0].Prompt, "Do not modify .noodle/mise.json.") {
		t.Fatalf("spawn prompt missing mise immutability note: %q", sp.calls[0].Prompt)
	}
	if !strings.Contains(sp.calls[0].Prompt, "Operate fully autonomously. Never ask the user questions.") {
		t.Fatalf("spawn prompt missing autonomous mode note: %q", sp.calls[0].Prompt)
	}
	if !strings.Contains(
		sp.calls[0].Prompt,
		"You may synthesize new queue items that are not present in mise.json when enforcing stage transitions",
	) {
		t.Fatalf("spawn prompt missing synthesized-item guidance: %q", sp.calls[0].Prompt)
	}
	if strings.Contains(sp.calls[0].Prompt, "mise.json schema (JSON):") {
		t.Fatalf("spawn prompt must not include mise schema: %q", sp.calls[0].Prompt)
	}
	if !sp.calls[0].AllowPrimaryCheckout {
		t.Fatal("expected prioritize spawn to allow primary checkout")
	}
	if sp.calls[0].WorktreePath != projectDir {
		t.Fatalf("worktree path = %q, want %q", sp.calls[0].WorktreePath, projectDir)
	}
	if len(wt.created) != 0 {
		t.Fatalf("unexpected worktree creates: %#v", wt.created)
	}

	updated, err := readQueue(queuePath)
	if err != nil {
		t.Fatalf("read queue: %v", err)
	}
	if len(updated.Items) != 1 || updated.Items[0].ID != PrioritizeTaskKey() {
		t.Fatalf("expected prioritize queue bootstrap item, got %#v", updated.Items)
	}
	if updated.Items[0].Skill != "prioritize" {
		t.Fatalf("queue skill = %q", updated.Items[0].Skill)
	}
}

func TestProcessControlCommandsPauseAndAck(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	control := filepath.Join(runtimeDir, "control.ndjson")
	if err := os.WriteFile(control, []byte(`{"id":"cmd-1","action":"pause"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write control: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Spawner:   &fakeSpawner{},
		Worktree:  &fakeWorktree{},
		Adapter:   &fakeAdapterRunner{},
		Mise:      &fakeMise{},
		Monitor:   fakeMonitor{},
		Registry:  testLoopRegistry(),
		Now:       func() time.Time { return time.Date(2026, 2, 22, 23, 0, 0, 0, time.UTC) },
		QueueFile: filepath.Join(runtimeDir, "queue.json"),
	})
	if err := l.processControlCommands(); err != nil {
		t.Fatalf("process commands: %v", err)
	}
	if l.state != StatePaused {
		t.Fatalf("state = %s", l.state)
	}

	ackPath := filepath.Join(runtimeDir, "control-ack.ndjson")
	data, err := os.ReadFile(ackPath)
	if err != nil {
		t.Fatalf("read ack file: %v", err)
	}
	var ack ControlAck
	if err := json.Unmarshal(data[:len(data)-1], &ack); err != nil {
		t.Fatalf("parse ack: %v", err)
	}
	if ack.ID != "cmd-1" || ack.Status != "ok" {
		t.Fatalf("ack = %#v", ack)
	}
}

func TestRetryLimitMarksFailedAndPreventsRespawn(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	review := false
	queuePath := filepath.Join(runtimeDir, "queue.json")
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-sonnet-4-6", Review: &review}}}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Recovery.MaxRetries = 0

	sp := &fakeSpawner{}
	wt := &fakeWorktree{}
	l := New(projectDir, "noodle", cfg, Dependencies{
		Spawner:   sp,
		Worktree:  wt,
		Adapter:   &fakeAdapterRunner{},
		Mise:      &fakeMise{},
		Monitor:   fakeMonitor{},
		Registry:  testLoopRegistry(),
		Now:       time.Now,
		QueueFile: queuePath,
	})
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("spawn cycle: %v", err)
	}
	if len(sp.sessions) != 1 {
		t.Fatalf("sessions = %d", len(sp.sessions))
	}
	sp.sessions[0].status = "failed"
	close(sp.sessions[0].done)

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("failure cycle: %v", err)
	}
	for _, call := range sp.calls[1:] {
		if call.Name == "42" {
			t.Fatalf("expected no respawn for failed item 42, calls = %#v", sp.calls)
		}
	}
	if _, ok := l.failedTargets["42"]; !ok {
		t.Fatal("expected target to be marked failed")
	}
	if _, err := os.Stat(filepath.Join(runtimeDir, "failed.json")); err != nil {
		t.Fatalf("expected failed.json: %v", err)
	}

	parsed, err := readQueue(queuePath)
	if err != nil {
		t.Fatalf("read queue: %v", err)
	}
	if len(parsed.Items) != 1 || parsed.Items[0].ID != PrioritizeTaskKey() {
		t.Fatalf("expected prioritize bootstrap item after max retries, got %#v", parsed.Items)
	}
}

func TestExitedStatusCountsAsFailureForPrioritize(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	queuePath := filepath.Join(runtimeDir, "queue.json")
	if err := writeQueueAtomic(queuePath, Queue{}); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Recovery.MaxRetries = 0

	sp := &fakeSpawner{}
	l := New(projectDir, "noodle", cfg, Dependencies{
		Spawner:   sp,
		Worktree:  &fakeWorktree{},
		Adapter:   &fakeAdapterRunner{},
		Mise:      &fakeMise{},
		Monitor:   fakeMonitor{},
		Registry:  testLoopRegistry(),
		Now:       time.Now,
		QueueFile: queuePath,
	})
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("spawn cycle: %v", err)
	}
	if len(sp.sessions) != 1 {
		t.Fatalf("sessions = %d", len(sp.sessions))
	}
	sp.sessions[0].status = "exited"
	close(sp.sessions[0].done)

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle with exited prioritize session: %v", err)
	}
	if l.runtimeRepairInFlight == nil {
		t.Fatal("expected runtime repair to be scheduled")
	}
	if len(sp.calls) != 2 {
		t.Fatalf("spawn calls = %d, want 2", len(sp.calls))
	}
	if !strings.HasPrefix(sp.calls[1].Name, "repair-runtime-") {
		t.Fatalf("expected repair spawn, got %q", sp.calls[1].Name)
	}
}

func TestSteerPrioritizeRegeneratesQueueWithPromptRationale(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Spawner:  &fakeSpawner{},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise: &fakeMise{brief: mise.Brief{
			Backlog: []adapter.BacklogItem{{ID: "1", Title: "Fix", Status: adapter.BacklogStatusOpen}},
		}},
		Monitor:   fakeMonitor{},
		Registry:  testLoopRegistry(),
		Now:       time.Now,
		QueueFile: queuePath,
	})

	if err := l.steer(PrioritizeTaskKey(), "prioritize security tasks"); err != nil {
		t.Fatalf("steer prioritize: %v", err)
	}
	queue, err := readQueue(queuePath)
	if err != nil {
		t.Fatalf("read queue: %v", err)
	}
	if len(queue.Items) != 1 {
		t.Fatalf("queue items = %d", len(queue.Items))
	}
	if queue.Items[0].ID != PrioritizeTaskKey() {
		t.Fatalf("unexpected id: %q", queue.Items[0].ID)
	}
	if queue.Items[0].Skill != "prioritize" {
		t.Fatalf("unexpected skill: %q", queue.Items[0].Skill)
	}
	if queue.Items[0].Title == "Fix" {
		t.Fatalf("expected prioritize bootstrap item, got backlog item title %q", queue.Items[0].Title)
	}
	if queue.Items[0].Rationale != "Chef steer: prioritize security tasks" {
		t.Fatalf("unexpected rationale: %q", queue.Items[0].Rationale)
	}
}

func TestCookBaseNameIncludesIDAndShortTitle(t *testing.T) {
	name := cookBaseName(QueueItem{
		ID:    "42",
		Title: "Refactor queue generation for reliability and clarity",
	})
	if !strings.HasPrefix(name, "42-refactor-queue-generation") {
		t.Fatalf("unexpected cook name: %q", name)
	}
	if len(name) > 64 {
		t.Fatalf("cook name too long: %d", len(name))
	}
}

func TestCookBaseNameFallsBackToIDWithoutTitle(t *testing.T) {
	name := cookBaseName(QueueItem{ID: "42", Title: ""})
	if name != "42" {
		t.Fatalf("cook name = %q", name)
	}
}

func TestReadQualityVerdictFromCanonical(t *testing.T) {
	path := filepath.Join(t.TempDir(), "canonical.ndjson")
	content := `{"provider":"claude","type":"action","message":"text: {\"accept\":false,\"feedback\":\"needs tests\"}","timestamp":"2026-02-22T20:00:00Z"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write canonical: %v", err)
	}
	verdict, found, err := readQualityVerdict(path)
	if err != nil {
		t.Fatalf("read verdict: %v", err)
	}
	if !found {
		t.Fatal("expected verdict to be found")
	}
	if verdict.Accept {
		t.Fatalf("expected reject verdict: %#v", verdict)
	}
	if verdict.Feedback != "needs tests" {
		t.Fatalf("feedback = %q", verdict.Feedback)
	}
}

func TestReadSessionTargetAcceptsRichIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prompt.txt")
	content := "Work backlog item plan/phase_02-ticket.7\n\nContext: test"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	target := readSessionTarget(path)
	if target != "plan/phase_02-ticket.7" {
		t.Fatalf("target = %q", target)
	}
}

func TestReadSessionTargetDetectsPrioritizePrompt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prompt.txt")
	content := "Use Skill(prioritize) to refresh .noodle/queue.json from .noodle/mise.json."
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	target := readSessionTarget(path)
	if target != PrioritizeTaskKey() {
		t.Fatalf("target = %q", target)
	}
}

func TestRunQualityCancelsSpawnedSessionOnContextDone(t *testing.T) {
	projectDir := t.TempDir()
	queuePath := filepath.Join(projectDir, ".noodle", "queue.json")
	if err := os.MkdirAll(filepath.Dir(queuePath), 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	sp := &fakeSpawner{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Spawner:   sp,
		Worktree:  &fakeWorktree{},
		Adapter:   &fakeAdapterRunner{},
		Mise:      &fakeMise{},
		Monitor:   fakeMonitor{},
		Registry:  testLoopRegistry(),
		Now:       time.Now,
		QueueFile: queuePath,
	})
	cook := &activeCook{
		queueItem: QueueItem{
			ID:       "42",
			Provider: "claude",
			Model:    "claude-sonnet-4-6",
		},
		worktreeName: "42-worktree",
		worktreePath: filepath.Join(projectDir, ".worktrees", "42-worktree"),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	accepted, feedback := l.runQuality(ctx, cook)
	if accepted {
		t.Fatal("expected canceled quality review run to be rejected")
	}
	if !strings.Contains(strings.ToLower(feedback), "canceled") {
		t.Fatalf("feedback = %q", feedback)
	}
	if len(sp.sessions) != 1 {
		t.Fatalf("quality spawn sessions = %d", len(sp.sessions))
	}
	if sp.sessions[0].status != "killed" {
		t.Fatalf("expected canceled quality session to be killed, got %q", sp.sessions[0].status)
	}
}

func TestTmuxSessionNameMatchesSanitizedLength(t *testing.T) {
	sessionID := strings.Repeat("A", 80) + "-with spaces"
	name := tmuxSessionName(sessionID)
	if !strings.HasPrefix(name, "noodle-") {
		t.Fatalf("unexpected prefix: %q", name)
	}
	token := strings.TrimPrefix(name, "noodle-")
	if len(token) > 48 {
		t.Fatalf("token too long: %d", len(token))
	}
}

func TestCycleRemovesStaleAdoptedSlotsBeforeScheduling(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions", "stale-session"), 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, "sessions", "stale-session", "meta.json"),
		[]byte(`{"status":"exited"}`),
		0o644,
	); err != nil {
		t.Fatalf("write meta: %v", err)
	}

	review := false
	queuePath := filepath.Join(runtimeDir, "queue.json")
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-sonnet-4-6", Review: &review}}}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Concurrency.MaxCooks = 1
	sp := &fakeSpawner{}
	l := New(projectDir, "noodle", cfg, Dependencies{
		Spawner:   sp,
		Worktree:  &fakeWorktree{},
		Adapter:   &fakeAdapterRunner{},
		Mise:      &fakeMise{},
		Monitor:   fakeMonitor{},
		Registry:  testLoopRegistry(),
		Now:       time.Now,
		QueueFile: queuePath,
	})
	l.adoptedTargets = map[string]string{"legacy-1": "stale-session"}

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}
	if len(sp.calls) != 1 {
		t.Fatalf("spawn calls = %d", len(sp.calls))
	}
	if len(l.adoptedTargets) != 0 {
		t.Fatalf("expected stale adopted target to be removed, got %#v", l.adoptedTargets)
	}
}

func TestBuildAdoptedCookDisablesReviewForPrioritize(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	queue := Queue{Items: []QueueItem{prioritizeQueueItem(config.DefaultConfig(), "")}}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Review.Enabled = true
	l := New(projectDir, "noodle", cfg, Dependencies{
		Spawner:   &fakeSpawner{},
		Worktree:  &fakeWorktree{},
		Adapter:   &fakeAdapterRunner{},
		Mise:      &fakeMise{},
		Monitor:   fakeMonitor{},
		Registry:  testLoopRegistry(),
		Now:       time.Now,
		QueueFile: queuePath,
	})

	cook, ok, err := l.buildAdoptedCook(PrioritizeTaskKey(), "session-1", "running")
	if err != nil {
		t.Fatalf("build adopted cook: %v", err)
	}
	if !ok {
		t.Fatal("expected adopted prioritize cook")
	}
	if cook.reviewEnabled {
		t.Fatal("expected prioritize adopted cook to keep review disabled")
	}
}

func TestCycleCompletesAdoptedCookFromMetaState(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	sessionID := "adopted-session"
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions", sessionID), 0o755); err != nil {
		t.Fatalf("mkdir session: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, "sessions", sessionID, "meta.json"),
		[]byte(`{"status":"completed"}`),
		0o644,
	); err != nil {
		t.Fatalf("write meta: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, "sessions", sessionID, "prompt.txt"),
		[]byte("Work backlog item 42\n"),
		0o644,
	); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	worktreePath := filepath.Join(projectDir, ".worktrees", "42")
	if err := os.WriteFile(
		filepath.Join(runtimeDir, "sessions", sessionID, "spawn.json"),
		[]byte(`{"worktree_path":"`+worktreePath+`"}`),
		0o644,
	); err != nil {
		t.Fatalf("write spawn metadata: %v", err)
	}

	review := false
	queuePath := filepath.Join(runtimeDir, "queue.json")
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-sonnet-4-6", Review: &review}}}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	wt := &fakeWorktree{}
	ar := &fakeAdapterRunner{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Spawner:   &fakeSpawner{},
		Worktree:  wt,
		Adapter:   ar,
		Mise:      &fakeMise{},
		Monitor:   fakeMonitor{},
		Registry:  testLoopRegistry(),
		Now:       time.Now,
		QueueFile: queuePath,
	})
	l.adoptedTargets = map[string]string{"42": sessionID}
	l.adoptedSessions = []string{sessionID}

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}
	if len(wt.merged) != 1 || wt.merged[0] != "42" {
		t.Fatalf("unexpected merged worktrees: %#v", wt.merged)
	}
	if len(ar.doneCalls) != 1 || ar.doneCalls[0] != "42" {
		t.Fatalf("unexpected done calls: %#v", ar.doneCalls)
	}
	if len(l.adoptedTargets) != 0 {
		t.Fatalf("expected adopted targets to be cleared, got %#v", l.adoptedTargets)
	}
	updated, err := readQueue(queuePath)
	if err != nil {
		t.Fatalf("read queue after adopted completion: %v", err)
	}
	if len(updated.Items) != 1 || updated.Items[0].ID != PrioritizeTaskKey() {
		t.Fatalf("expected prioritize bootstrap item after adopted completion, got %#v", updated.Items)
	}
}

func TestCycleSchedulesRuntimeRepairForMiseErrors(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")

	sp := &fakeSpawner{}
	wt := &fakeWorktree{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Spawner:   sp,
		Worktree:  wt,
		Adapter:   &fakeAdapterRunner{},
		Mise:      &fakeMise{err: errors.New("plans sync failed")},
		Monitor:   fakeMonitor{},
		Registry:  testLoopRegistry(),
		Now:       time.Now,
		QueueFile: queuePath,
	})
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}
	if len(sp.calls) != 1 {
		t.Fatalf("repair spawn calls = %d", len(sp.calls))
	}
	if sp.calls[0].Skill != "oops" {
		t.Fatalf("repair skill = %q", sp.calls[0].Skill)
	}
	if !strings.HasPrefix(sp.calls[0].Name, "repair-runtime-") {
		t.Fatalf("unexpected repair name: %q", sp.calls[0].Name)
	}
	if l.runtimeRepairInFlight == nil {
		t.Fatal("expected runtime repair to be in-flight")
	}
	if len(wt.created) != 1 {
		t.Fatalf("repair worktree create calls = %d", len(wt.created))
	}
}

func TestCycleResumesSchedulingAfterRepairCompletion(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	review := false
	queuePath := filepath.Join(runtimeDir, "queue.json")
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-sonnet-4-6", Review: &review}}}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	sp := &fakeSpawner{}
	miseBuilder := &fakeMise{
		results: []fakeMiseResult{
			{err: errors.New("backlog sync failed")},
			{brief: mise.Brief{}},
		},
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Spawner:   sp,
		Worktree:  &fakeWorktree{},
		Adapter:   &fakeAdapterRunner{},
		Mise:      miseBuilder,
		Monitor:   fakeMonitor{},
		Registry:  testLoopRegistry(),
		Now:       time.Now,
		QueueFile: queuePath,
	})

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("first cycle: %v", err)
	}
	if len(sp.sessions) != 1 {
		t.Fatalf("sessions after repair spawn = %d", len(sp.sessions))
	}
	sp.sessions[0].status = "completed"
	close(sp.sessions[0].done)

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("second cycle: %v", err)
	}
	if len(sp.calls) != 2 {
		t.Fatalf("expected repair + cook spawns, got %d", len(sp.calls))
	}
	if sp.calls[1].Name != "42" {
		t.Fatalf("expected cook spawn after repair, got %q", sp.calls[1].Name)
	}
	if l.runtimeRepairInFlight != nil {
		t.Fatal("expected runtime repair to be cleared")
	}
}

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
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/internal/statusfile"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/monitor"
	"github.com/poteto/noodle/worktree"
)

type fakeSession struct {
	id     string
	status string
	done   chan struct{}
}

func (s *fakeSession) ID() string     { return s.id }
func (s *fakeSession) Status() string { return s.status }
func (s *fakeSession) Events() <-chan dispatcher.SessionEvent {
	return make(chan dispatcher.SessionEvent)
}
func (s *fakeSession) Done() <-chan struct{} { return s.done }
func (s *fakeSession) TotalCost() float64    { return 0 }
func (s *fakeSession) Kill() error           { s.status = "killed"; close(s.done); return nil }

type fakeDispatcher struct {
	calls       []dispatcher.DispatchRequest
	sessions    []*fakeSession
	dispatchErr error
}

func (f *fakeDispatcher) Dispatch(_ context.Context, req dispatcher.DispatchRequest) (dispatcher.Session, error) {
	f.calls = append(f.calls, req)
	if f.dispatchErr != nil {
		return nil, f.dispatchErr
	}
	s := &fakeSession{id: req.Name + "-id", status: "running", done: make(chan struct{})}
	f.sessions = append(f.sessions, s)
	return s, nil
}

type fakeWorktree struct {
	created         []string
	merged          []string
	remoteMerged    []string
	cleaned         []string
	mergeErr        error
	remoteMergeErr  error
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
	return f.mergeErr
}
func (f *fakeWorktree) MergeRemoteBranch(branch string) error {
	f.remoteMerged = append(f.remoteMerged, branch)
	return f.remoteMergeErr
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
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-opus-4-6"}}}
	if err := writeQueueAtomic(filepath.Join(runtimeDir, "queue.json"), queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	sp := &fakeDispatcher{}
	wt := &fakeWorktree{}
	ar := &fakeAdapterRunner{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: sp,
		Worktree:   wt,
		Adapter:    ar,
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  filepath.Join(runtimeDir, "queue.json"),
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
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-opus-4-6"}}}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	existingWorktree := filepath.Join(projectDir, ".worktrees", "42")
	if err := os.MkdirAll(existingWorktree, 0o755); err != nil {
		t.Fatalf("mkdir existing worktree: %v", err)
	}

	sp := &fakeDispatcher{}
	wt := &fakeWorktree{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: sp,
		Worktree:   wt,
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
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
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-opus-4-6"}}}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	existingWorktree := filepath.Join(projectDir, ".worktrees", "42")
	sp := &fakeDispatcher{}
	wt := &fakeWorktree{createErr: errors.New("worktree '42' already exists at " + existingWorktree)}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: sp,
		Worktree:   wt,
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
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
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-opus-4-6"}}}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(projectDir, ".worktrees", "42"), 0o755); err != nil {
		t.Fatalf("mkdir existing worktree: %v", err)
	}

	sp := &fakeDispatcher{dispatchErr: errors.New("spawn failed")}
	wt := &fakeWorktree{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: sp,
		Worktree:   wt,
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
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
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-opus-4-6"}}}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	sp := &fakeDispatcher{dispatchErr: errors.New("spawn failed")}
	wt := &fakeWorktree{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: sp,
		Worktree:   wt,
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
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
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-opus-4-6"}}}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	sp := &fakeDispatcher{}
	wt := &fakeWorktree{}
	ar := &fakeAdapterRunner{}
	briefWithPlans := mise.Brief{Plans: []mise.PlanSummary{{ID: 1, Status: "open"}}}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: sp,
		Worktree:   wt,
		Adapter:    ar,
		Mise:       &fakeMise{brief: briefWithPlans},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
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

func TestCycleEntersIdleWhenNoPlansRemain(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")

	sp := &fakeDispatcher{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: sp,
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
	})
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	if l.state != StateIdle {
		t.Fatalf("state = %s, want idle", l.state)
	}
	if len(sp.calls) != 0 {
		t.Fatalf("expected no spawn calls when idle, got %d", len(sp.calls))
	}

	statusPath := filepath.Join(runtimeDir, "status.json")
	status, err := statusfile.Read(statusPath)
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status.LoopState != "idle" {
		t.Fatalf("status loop_state = %q, want idle", status.LoopState)
	}
}

func TestCycleIdleWakesWhenPlansAppear(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")

	fm := &fakeMise{}
	sp := &fakeDispatcher{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: sp,
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       fm,
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
	})

	// First cycle: no plans → idle
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if l.state != StateIdle {
		t.Fatalf("state after cycle 1 = %s, want idle", l.state)
	}

	// Simulate new plans appearing
	fm.brief = mise.Brief{Plans: []mise.PlanSummary{{ID: 1, Status: "open"}}}

	// Second cycle: idle → running, bootstraps prioritize
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	if l.state != StateRunning {
		t.Fatalf("state after cycle 2 = %s, want running", l.state)
	}
	if len(sp.calls) != 1 {
		t.Fatalf("expected 1 spawn call after wake, got %d", len(sp.calls))
	}
	if sp.calls[0].Skill != "prioritize" {
		t.Fatalf("expected prioritize spawn, got skill %q", sp.calls[0].Skill)
	}
}

func TestCycleBootstrapsPrioritizeUsesRegistrySkill(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")

	sp := &fakeDispatcher{}
	wt := &fakeWorktree{}
	briefWithPlans := mise.Brief{Plans: []mise.PlanSummary{{ID: 1, Status: "open"}}}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: sp,
		Worktree:   wt,
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{brief: briefWithPlans},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
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
	if !strings.Contains(sp.calls[0].Prompt, "Use Skill(prioritize) to refresh the queue from .noodle/mise.json.") {
		t.Fatalf("spawn prompt missing skill invocation: %q", sp.calls[0].Prompt)
	}
	if !strings.Contains(sp.calls[0].Prompt, "queue-next.json") {
		t.Fatalf("spawn prompt missing queue-next.json instruction: %q", sp.calls[0].Prompt)
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
		"You may synthesize queue items for non-execute task types",
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
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        func() time.Time { return time.Date(2026, 2, 22, 23, 0, 0, 0, time.UTC) },
		QueueFile:  filepath.Join(runtimeDir, "queue.json"),
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
	queuePath := filepath.Join(runtimeDir, "queue.json")
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-opus-4-6"}}}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Recovery.MaxRetries = 0

	briefWithPlans := mise.Brief{Plans: []mise.PlanSummary{{ID: 42, Status: "open"}}}
	sp := &fakeDispatcher{}
	wt := &fakeWorktree{}
	l := New(projectDir, "noodle", cfg, Dependencies{
		Dispatcher: sp,
		Worktree:   wt,
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{brief: briefWithPlans},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
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

	briefWithPlans := mise.Brief{Plans: []mise.PlanSummary{{ID: 1, Status: "open"}}}
	sp := &fakeDispatcher{}
	l := New(projectDir, "noodle", cfg, Dependencies{
		Dispatcher: sp,
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{brief: briefWithPlans},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
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
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
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

func TestCycleRegistryErrorBlocksOnce(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	// Create a loop with a registry error (simulates discovery failure)
	l := &Loop{
		projectDir:            projectDir,
		runtimeDir:            runtimeDir,
		registryErr:           errors.New("task type discovery failed: bad frontmatter"),
		activeByTarget:        map[string]*activeCook{},
		activeByID:            map[string]*activeCook{},
		adoptedTargets:        map[string]string{},
		failedTargets:         map[string]string{},
		processedIDs:          map[string]struct{}{},
		runtimeRepairAttempts: map[string]int{},
		deps: Dependencies{
			Mise:    &fakeMise{brief: mise.Brief{}},
			Monitor: fakeMonitor{},
		},
	}

	// Cycle (the --once path) should fail with the registry error
	err := l.Cycle(context.Background())
	if err == nil {
		t.Fatal("expected Cycle to return registry error")
	}
	if !strings.Contains(err.Error(), "task type discovery failed") {
		t.Fatalf("wrong error: %v", err)
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
	content := "Use Skill(prioritize) to refresh the queue from .noodle/mise.json."
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	target := readSessionTarget(path)
	if target != PrioritizeTaskKey() {
		t.Fatalf("target = %q", target)
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
	queuePath := filepath.Join(runtimeDir, "queue.json")
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-opus-4-6"}}}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Concurrency.MaxCooks = 1
	sp := &fakeDispatcher{}
	l := New(projectDir, "noodle", cfg, Dependencies{
		Dispatcher: sp,
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
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

func TestCycleStampsLoopStateWhenPaused(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-opus-4-6"}}}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Autonomy = "approve"
	l := New(projectDir, "noodle", cfg, Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
	})
	l.state = StatePaused

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	statusPath := filepath.Join(runtimeDir, "status.json")
	status, err := statusfile.Read(statusPath)
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status.Autonomy != "approve" {
		t.Fatalf("autonomy = %q, want approve", status.Autonomy)
	}
}

func TestCycleStampsLoopStateWhenDraining(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-opus-4-6"}}}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Autonomy = "auto"
	l := New(projectDir, "noodle", cfg, Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
	})
	l.state = StateDraining

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	statusPath := filepath.Join(runtimeDir, "status.json")
	status, err := statusfile.Read(statusPath)
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status.Autonomy != "auto" {
		t.Fatalf("autonomy = %q, want auto", status.Autonomy)
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
	queuePath := filepath.Join(runtimeDir, "queue.json")
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-opus-4-6"}}}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	briefWithPlans := mise.Brief{Plans: []mise.PlanSummary{{ID: 42, Status: "open"}}}
	wt := &fakeWorktree{}
	ar := &fakeAdapterRunner{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   wt,
		Adapter:    ar,
		Mise:       &fakeMise{brief: briefWithPlans},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
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

func TestMergeCookUsesRemoteBranchSyncResult(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	sessionID := "session-a"
	sessionPath := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionPath, 0o755); err != nil {
		t.Fatalf("mkdir session path: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(sessionPath, "spawn.json"),
		[]byte(`{"sync":{"type":"branch","branch":"noodle/session-a"}}`),
		0o644,
	); err != nil {
		t.Fatalf("write spawn metadata: %v", err)
	}

	queuePath := filepath.Join(runtimeDir, "queue.json")
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-sonnet-4-6"}}}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	wt := &fakeWorktree{}
	ar := &fakeAdapterRunner{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   wt,
		Adapter:    ar,
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
	})

	if err := l.mergeCook(
		context.Background(),
		QueueItem{ID: "42", Provider: "claude", Model: "claude-sonnet-4-6"},
		"42",
		sessionID,
	); err != nil {
		t.Fatalf("mergeCook: %v", err)
	}
	if len(wt.remoteMerged) != 1 || wt.remoteMerged[0] != "noodle/session-a" {
		t.Fatalf("unexpected remote merges: %#v", wt.remoteMerged)
	}
	if len(wt.merged) != 0 {
		t.Fatalf("expected no local merge, got %#v", wt.merged)
	}
}

func TestMergeCookFallsBackToLocalWorktreeMerge(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-sonnet-4-6"}}}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	wt := &fakeWorktree{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   wt,
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
	})

	if err := l.mergeCook(
		context.Background(),
		QueueItem{ID: "42", Provider: "claude", Model: "claude-sonnet-4-6"},
		"42",
		"",
	); err != nil {
		t.Fatalf("mergeCook: %v", err)
	}
	if len(wt.merged) != 1 || wt.merged[0] != "42" {
		t.Fatalf("unexpected local merges: %#v", wt.merged)
	}
	if len(wt.remoteMerged) != 0 {
		t.Fatalf("expected no remote merge, got %#v", wt.remoteMerged)
	}
}

func TestCycleMergeConflictMarksFailedAndSkipsWithoutRuntimeRepair(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	queue := Queue{Items: []QueueItem{{ID: "42", Provider: "claude", Model: "claude-sonnet-4-6"}}}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	briefWithPlans := mise.Brief{Plans: []mise.PlanSummary{{ID: 42, Status: "open"}}}
	sp := &fakeDispatcher{}
	wt := &fakeWorktree{
		remoteMergeErr: &worktree.MergeConflictError{Branch: "origin/noodle/session-a"},
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: sp,
		Worktree:   wt,
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{brief: briefWithPlans},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
	})
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("spawn cycle: %v", err)
	}
	if len(sp.sessions) != 1 {
		t.Fatalf("sessions = %d", len(sp.sessions))
	}

	sessionID := sp.sessions[0].id
	sessionPath := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionPath, 0o755); err != nil {
		t.Fatalf("mkdir session path: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(sessionPath, "spawn.json"),
		[]byte(`{"sync":{"type":"branch","branch":"noodle/session-a"}}`),
		0o644,
	); err != nil {
		t.Fatalf("write spawn metadata: %v", err)
	}

	sp.sessions[0].status = "completed"
	close(sp.sessions[0].done)

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("completion cycle: %v", err)
	}
	if l.runtimeRepairInFlight != nil {
		t.Fatalf("expected no runtime repair in flight, got %#v", l.runtimeRepairInFlight)
	}
	if reason, ok := l.failedTargets["42"]; !ok {
		t.Fatal("expected target to be marked failed")
	} else if !strings.Contains(reason, "merge conflict") {
		t.Fatalf("failed reason = %q", reason)
	}

	updated, err := readQueue(queuePath)
	if err != nil {
		t.Fatalf("read queue after conflict: %v", err)
	}
	if len(updated.Items) != 1 || updated.Items[0].ID != PrioritizeTaskKey() {
		t.Fatalf("expected prioritize bootstrap item after conflict, got %#v", updated.Items)
	}
}

func TestApprovalAutoCanMergeTrueAutoMerges(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	// execute task type has CanMerge=true (no Merge override in registry)
	queue := Queue{Items: []QueueItem{{ID: "42", TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6"}}}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Autonomy = "auto"

	sp := &fakeDispatcher{}
	wt := &fakeWorktree{}
	ar := &fakeAdapterRunner{}
	briefWithPlans := mise.Brief{Plans: []mise.PlanSummary{{ID: 1, Status: "open"}}}
	l := New(projectDir, "noodle", cfg, Dependencies{
		Dispatcher: sp,
		Worktree:   wt,
		Adapter:    ar,
		Mise:       &fakeMise{brief: briefWithPlans},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
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
		t.Fatalf("worktree merges = %d, want 1 (auto-merge)", len(wt.merged))
	}
	if len(l.pendingReview) != 0 {
		t.Fatalf("pendingReview should be empty, got %d item(s)", len(l.pendingReview))
	}
}

func TestApprovalAutoCanMergeFalseParks(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	// review task type has CanMerge=false (Merge: boolPtr(false) in registry)
	queue := Queue{Items: []QueueItem{{ID: "42", TaskKey: "review", Provider: "claude", Model: "claude-opus-4-6"}}}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Autonomy = "auto"

	sp := &fakeDispatcher{}
	wt := &fakeWorktree{}
	briefWithPlans := mise.Brief{Plans: []mise.PlanSummary{{ID: 1, Status: "open"}}}
	l := New(projectDir, "noodle", cfg, Dependencies{
		Dispatcher: sp,
		Worktree:   wt,
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{brief: briefWithPlans},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
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
	if len(wt.merged) != 0 {
		t.Fatalf("worktree merges = %d, want 0 (task disallows merge)", len(wt.merged))
	}
	if len(l.pendingReview) != 1 {
		t.Fatalf("pendingReview = %d, want 1", len(l.pendingReview))
	}
	items, err := ReadPendingReview(runtimeDir)
	if err != nil {
		t.Fatalf("ReadPendingReview: %v", err)
	}
	if len(items) != 1 || items[0].ID != "42" {
		t.Fatalf("pending review file items = %#v", items)
	}
}

func TestApprovalApproveCanMergeTrueParks(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	// execute task type has CanMerge=true, but autonomy=approve overrides
	queue := Queue{Items: []QueueItem{{ID: "42", TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6"}}}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Autonomy = "approve"

	sp := &fakeDispatcher{}
	wt := &fakeWorktree{}
	briefWithPlans := mise.Brief{Plans: []mise.PlanSummary{{ID: 1, Status: "open"}}}
	l := New(projectDir, "noodle", cfg, Dependencies{
		Dispatcher: sp,
		Worktree:   wt,
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{brief: briefWithPlans},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
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
	if len(wt.merged) != 0 {
		t.Fatalf("worktree merges = %d, want 0 (approve mode overrides)", len(wt.merged))
	}
	if len(l.pendingReview) != 1 {
		t.Fatalf("pendingReview = %d, want 1", len(l.pendingReview))
	}
	items, err := ReadPendingReview(runtimeDir)
	if err != nil {
		t.Fatalf("ReadPendingReview: %v", err)
	}
	if len(items) != 1 || items[0].ID != "42" {
		t.Fatalf("pending review file items = %#v", items)
	}
}

func TestApprovalApproveCanMergeFalseParks(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	// review task type has CanMerge=false AND autonomy=approve
	queue := Queue{Items: []QueueItem{{ID: "42", TaskKey: "review", Provider: "claude", Model: "claude-opus-4-6"}}}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Autonomy = "approve"

	sp := &fakeDispatcher{}
	wt := &fakeWorktree{}
	briefWithPlans := mise.Brief{Plans: []mise.PlanSummary{{ID: 1, Status: "open"}}}
	l := New(projectDir, "noodle", cfg, Dependencies{
		Dispatcher: sp,
		Worktree:   wt,
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{brief: briefWithPlans},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
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
	if len(wt.merged) != 0 {
		t.Fatalf("worktree merges = %d, want 0 (both canMerge=false and approve mode)", len(wt.merged))
	}
	if len(l.pendingReview) != 1 {
		t.Fatalf("pendingReview = %d, want 1", len(l.pendingReview))
	}
	items, err := ReadPendingReview(runtimeDir)
	if err != nil {
		t.Fatalf("ReadPendingReview: %v", err)
	}
	if len(items) != 1 || items[0].ID != "42" {
		t.Fatalf("pending review file items = %#v", items)
	}
}

// selectiveErrDispatcher fails on specific call indices, succeeds on others.
type selectiveErrDispatcher struct {
	calls    []dispatcher.DispatchRequest
	sessions []*fakeSession
	failAt   map[int]error // call index → error
}

func (d *selectiveErrDispatcher) Dispatch(_ context.Context, req dispatcher.DispatchRequest) (dispatcher.Session, error) {
	index := len(d.calls)
	d.calls = append(d.calls, req)
	if err, ok := d.failAt[index]; ok {
		return nil, err
	}
	s := &fakeSession{id: req.Name + "-id", status: "running", done: make(chan struct{})}
	d.sessions = append(d.sessions, s)
	return s, nil
}

// TestNoDoubleSpawnAfterFailedRetryRepair reproduces a bug where a queue item
// gets spawned fresh after a failed retry triggers runtime repair.
//
// Sequence:
//  1. Cycle 1: item "37" spawns session A
//  2. Session A fails (Done fires)
//  3. Cycle 2: collectCompleted picks up A, deletes from activeByTarget,
//     retryCook→spawnCook fails (dispatch error) → handleRuntimeIssue
//     starts repair → returns nil. activeByTarget["37"] is now EMPTY.
//  4. Repair completes
//  5. Cycle 3: planCycleSpawns sees "37" not in activeByTarget → spawns fresh
//
// This is a bug: the item's tracking is lost between the failed retry and
// repair completion, causing a duplicate spawn (attempt counter resets to 0).
// In production with tmux, if the previous session was still alive (false
// Done signal), two agents would run simultaneously for the same item.
func TestNoDoubleSpawnAfterFailedRetryRepair(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	queue := Queue{Items: []QueueItem{{
		ID:       "37",
		TaskKey:  "execute",
		Skill:    "execute",
		Provider: "claude",
		Model:    "claude-opus-4-6",
		Plan:     []string{"plans/37-test/overview"},
	}}}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	// Call 0: cycle 1 spawns "37" → succeeds
	// Call 1: cycle 2 retry of "37" → fails (triggers runtime repair)
	// Call 2: cycle 2 repair session → succeeds
	// Call 3+: cycle 3 onward → succeeds (bug: fresh spawn of "37")
	sp := &selectiveErrDispatcher{
		failAt: map[int]error{1: errors.New("tmux unavailable")},
	}
	wt := &fakeWorktree{}

	cfg := config.DefaultConfig()
	cfg.Recovery.MaxRetries = 3

	l := New(projectDir, "noodle", cfg, Dependencies{
		Dispatcher: sp,
		Worktree:   wt,
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{brief: mise.Brief{Plans: []mise.PlanSummary{{ID: 37, Status: "open", Title: "Test", Directory: "test"}}}},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
	})

	// --- Cycle 1: spawns session A for item "37" ---
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if len(sp.sessions) != 1 {
		t.Fatalf("expected 1 session after cycle 1, got %d", len(sp.sessions))
	}
	if _, ok := l.activeByTarget["37"]; !ok {
		t.Fatal("expected item 37 in activeByTarget after cycle 1")
	}

	// Simulate: session A fails
	sessionA := sp.sessions[0]
	sessionA.status = "failed"
	close(sessionA.done)

	// Create the worktree directory so ensureWorktree doesn't call Create again
	// (mirrors production: worktree exists from cycle 1's dispatch).
	if err := os.MkdirAll(filepath.Join(projectDir, ".worktrees", "37"), 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}

	// --- Cycle 2: collect A, retry fails, repair starts ---
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	if l.runtimeRepairInFlight == nil {
		t.Fatal("expected runtime repair in flight after cycle 2")
	}
	if _, ok := l.activeByTarget["37"]; ok {
		t.Log("activeByTarget still has 37 after failed retry — good (not the bug path)")
	}

	// Simulate: repair session completes
	repairSession := sp.sessions[len(sp.sessions)-1]
	repairSession.status = "completed"
	close(repairSession.done)

	// --- Cycle 3: repair clears, pending retry fires ---
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 3: %v", err)
	}

	// After repair, the pending retry should fire with the correct attempt
	// counter (1, not 0). The bug caused a fresh spawn via planCycleSpawns
	// with attempt 0 because activeByTarget lost tracking of the item.
	cook37, ok := l.activeByTarget["37"]
	if !ok {
		t.Fatal("expected item 37 in activeByTarget after cycle 3 (pending retry should have fired)")
	}
	if cook37.attempt == 0 {
		t.Errorf(
			"BUG: item '37' was spawned fresh (attempt 0) after repair "+
				"(attempt counter lost, activeByTarget tracking gap between "+
				"failed retry and repair completion)",
		)
	}
}

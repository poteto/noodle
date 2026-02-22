package loop

import (
	"context"
	"encoding/json"
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
}

func (f *fakeSpawner) Spawn(_ context.Context, req spawner.SpawnRequest) (spawner.Session, error) {
	f.calls = append(f.calls, req)
	s := &fakeSession{id: req.Name + "-id", status: "running", done: make(chan struct{})}
	f.sessions = append(f.sessions, s)
	return s, nil
}

type fakeWorktree struct {
	created []string
	merged  []string
	cleaned []string
}

func (f *fakeWorktree) Create(name string) error {
	f.created = append(f.created, name)
	return nil
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
	brief mise.Brief
}

func (f *fakeMise) Build(_ context.Context) (mise.Brief, []string, error) {
	return f.brief, nil, nil
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
	if len(updated.Items) != 0 {
		t.Fatalf("expected completed queue item to be removed, got %#v", updated.Items)
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
	if len(sp.calls) != 1 {
		t.Fatalf("expected no respawn, spawn calls = %d", len(sp.calls))
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
	if len(parsed.Items) != 0 {
		t.Fatalf("queue should be trimmed after max retries: %#v", parsed.Items)
	}
}

func TestSteerSousChefRegeneratesQueueWithPromptRationale(t *testing.T) {
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
		Now:       time.Now,
		QueueFile: queuePath,
	})

	if err := l.steer("sous-chef", "prioritize security tasks"); err != nil {
		t.Fatalf("steer sous-chef: %v", err)
	}
	queue, err := readQueue(queuePath)
	if err != nil {
		t.Fatalf("read queue: %v", err)
	}
	if len(queue.Items) != 1 {
		t.Fatalf("queue items = %d", len(queue.Items))
	}
	if queue.Items[0].Title != "Fix" {
		t.Fatalf("unexpected title: %q", queue.Items[0].Title)
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

func TestReadTasterVerdictFromCanonical(t *testing.T) {
	path := filepath.Join(t.TempDir(), "canonical.ndjson")
	content := `{"provider":"claude","type":"action","message":"text: {\"accept\":false,\"feedback\":\"needs tests\"}","timestamp":"2026-02-22T20:00:00Z"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write canonical: %v", err)
	}
	verdict, found, err := readTasterVerdict(path)
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
	if len(updated.Items) != 0 {
		t.Fatalf("expected queue to be empty after adopted completion, got %#v", updated.Items)
	}
}

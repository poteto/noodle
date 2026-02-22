package loop

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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

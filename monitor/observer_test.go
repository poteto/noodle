package monitor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTmuxObserverDetectsDeadPaneAndLogStats(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionID := "cook-a"
	sessionPath := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionPath, 0o755); err != nil {
		t.Fatalf("mkdir session path: %v", err)
	}

	canonicalPath := filepath.Join(sessionPath, "canonical.ndjson")
	content := []byte("{\"type\":\"init\",\"timestamp\":\"2026-02-22T15:00:00Z\"}\n")
	if err := os.WriteFile(canonicalPath, content, 0o644); err != nil {
		t.Fatalf("write canonical file: %v", err)
	}

	observer := NewTmuxObserver(runtimeDir)
	observer.run = func(name string, args ...string) error {
		if name != "tmux" {
			return fmt.Errorf("unexpected command %q", name)
		}
		return fmt.Errorf("session missing")
	}

	observation, err := observer.Observe(sessionID)
	if err != nil {
		t.Fatalf("observe session: %v", err)
	}
	if observation.Alive {
		t.Fatal("expected dead pane")
	}
	if observation.LogSize == 0 {
		t.Fatal("expected non-zero log size")
	}
	if observation.LogMTime.IsZero() {
		t.Fatal("expected log mtime")
	}
	if observation.LogMTime.After(time.Now().Add(1 * time.Second)) {
		t.Fatalf("unexpected mtime in future: %v", observation.LogMTime)
	}
}

func TestHeartbeatObserverAliveWhenFresh(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionID := "cook-a"
	sessionPath := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionPath, 0o755); err != nil {
		t.Fatalf("mkdir session path: %v", err)
	}

	now := time.Date(2026, time.February, 24, 6, 0, 0, 0, time.UTC)
	payload, err := json.Marshal(map[string]any{
		"timestamp":   now.Add(-10 * time.Second).Format(time.RFC3339Nano),
		"ttl_seconds": 30,
	})
	if err != nil {
		t.Fatalf("marshal heartbeat: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionPath, "heartbeat.json"), payload, 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	observer := NewHeartbeatObserver(runtimeDir)
	observer.now = func() time.Time { return now }
	observation, err := observer.Observe(sessionID)
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if !observation.Alive {
		t.Fatal("expected alive heartbeat")
	}
}

func TestHeartbeatObserverNotAliveWhenStale(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionID := "cook-a"
	sessionPath := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionPath, 0o755); err != nil {
		t.Fatalf("mkdir session path: %v", err)
	}

	now := time.Date(2026, time.February, 24, 6, 0, 0, 0, time.UTC)
	payload, err := json.Marshal(map[string]any{
		"timestamp":   now.Add(-61 * time.Second).Format(time.RFC3339Nano),
		"ttl_seconds": 30,
	})
	if err != nil {
		t.Fatalf("marshal heartbeat: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionPath, "heartbeat.json"), payload, 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	observer := NewHeartbeatObserver(runtimeDir)
	observer.now = func() time.Time { return now }
	observation, err := observer.Observe(sessionID)
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if observation.Alive {
		t.Fatal("expected stale heartbeat to be not alive")
	}
}

func TestHeartbeatObserverMissingHeartbeatIsNotAlive(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionID := "cook-a"
	sessionPath := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionPath, 0o755); err != nil {
		t.Fatalf("mkdir session path: %v", err)
	}

	observer := NewHeartbeatObserver(runtimeDir)
	observation, err := observer.Observe(sessionID)
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if observation.Alive {
		t.Fatal("expected missing heartbeat to be not alive")
	}
}

type countingObserver struct {
	calls []string
}

func (o *countingObserver) Observe(sessionID string) (Observation, error) {
	o.calls = append(o.calls, sessionID)
	return Observation{SessionID: sessionID}, nil
}

func TestCompositeObserverRoutesByRuntime(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionTmux := "cook-local"
	sessionRemote := "cook-remote"
	for _, sessionID := range []string{sessionTmux, sessionRemote} {
		if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions", sessionID), 0o755); err != nil {
			t.Fatalf("mkdir session path: %v", err)
		}
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, "sessions", sessionTmux, "spawn.json"),
		[]byte(`{"runtime":"tmux"}`),
		0o644,
	); err != nil {
		t.Fatalf("write tmux spawn: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, "sessions", sessionRemote, "spawn.json"),
		[]byte(`{"runtime":"sprites"}`),
		0o644,
	); err != nil {
		t.Fatalf("write remote spawn: %v", err)
	}

	tmuxObserver := &countingObserver{}
	remoteObserver := &countingObserver{}
	composite := NewCompositeObserver(runtimeDir, tmuxObserver, remoteObserver)

	if _, err := composite.Observe(sessionTmux); err != nil {
		t.Fatalf("observe tmux: %v", err)
	}
	if _, err := composite.Observe(sessionRemote); err != nil {
		t.Fatalf("observe remote: %v", err)
	}

	if len(tmuxObserver.calls) != 1 || tmuxObserver.calls[0] != sessionTmux {
		t.Fatalf("tmux observer calls = %#v", tmuxObserver.calls)
	}
	if len(remoteObserver.calls) != 1 || remoteObserver.calls[0] != sessionRemote {
		t.Fatalf("remote observer calls = %#v", remoteObserver.calls)
	}
}

func TestCompositeObserverCachesRoutingDecision(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionID := "cook-remote"
	sessionPath := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionPath, 0o755); err != nil {
		t.Fatalf("mkdir session path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionPath, "spawn.json"), []byte(`{"runtime":"sprites"}`), 0o644); err != nil {
		t.Fatalf("write spawn: %v", err)
	}

	tmuxObserver := &countingObserver{}
	remoteObserver := &countingObserver{}
	composite := NewCompositeObserver(runtimeDir, tmuxObserver, remoteObserver)

	if _, err := composite.Observe(sessionID); err != nil {
		t.Fatalf("first observe: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionPath, "spawn.json"), []byte(`{"runtime":"tmux"}`), 0o644); err != nil {
		t.Fatalf("rewrite spawn: %v", err)
	}
	if _, err := composite.Observe(sessionID); err != nil {
		t.Fatalf("second observe: %v", err)
	}

	if len(remoteObserver.calls) != 2 {
		t.Fatalf("remote observer calls = %#v, want 2", remoteObserver.calls)
	}
	if len(tmuxObserver.calls) != 0 {
		t.Fatalf("tmux observer calls = %#v, want none", tmuxObserver.calls)
	}
}

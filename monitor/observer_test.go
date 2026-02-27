package monitor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
	sessionLocal := "cook-local"
	sessionRemote := "cook-remote"
	for _, sessionID := range []string{sessionLocal, sessionRemote} {
		if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions", sessionID), 0o755); err != nil {
			t.Fatalf("mkdir session path: %v", err)
		}
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, "sessions", sessionLocal, "spawn.json"),
		[]byte(`{"runtime":"process"}`),
		0o644,
	); err != nil {
		t.Fatalf("write local spawn: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, "sessions", sessionRemote, "spawn.json"),
		[]byte(`{"runtime":"sprites"}`),
		0o644,
	); err != nil {
		t.Fatalf("write remote spawn: %v", err)
	}

	localObserver := &countingObserver{}
	remoteObserver := &countingObserver{}
	composite := NewCompositeObserver(runtimeDir, localObserver, remoteObserver)

	if _, err := composite.Observe(sessionLocal); err != nil {
		t.Fatalf("observe local: %v", err)
	}
	if _, err := composite.Observe(sessionRemote); err != nil {
		t.Fatalf("observe remote: %v", err)
	}

	if len(localObserver.calls) != 1 || localObserver.calls[0] != sessionLocal {
		t.Fatalf("local observer calls = %#v", localObserver.calls)
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

	localObserver := &countingObserver{}
	remoteObserver := &countingObserver{}
	composite := NewCompositeObserver(runtimeDir, localObserver, remoteObserver)

	if _, err := composite.Observe(sessionID); err != nil {
		t.Fatalf("first observe: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionPath, "spawn.json"), []byte(`{"runtime":"process"}`), 0o644); err != nil {
		t.Fatalf("rewrite spawn: %v", err)
	}
	if _, err := composite.Observe(sessionID); err != nil {
		t.Fatalf("second observe: %v", err)
	}

	if len(remoteObserver.calls) != 2 {
		t.Fatalf("remote observer calls = %#v, want 2", remoteObserver.calls)
	}
	if len(localObserver.calls) != 0 {
		t.Fatalf("local observer calls = %#v, want none", localObserver.calls)
	}
}

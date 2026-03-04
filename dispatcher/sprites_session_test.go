package dispatcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/parse"
)

func TestSpritesSessionResolveAndMarkDoneCompletedFromResult(t *testing.T) {
	session := newSpritesSession(spritesSessionConfig{
		id:            "session-a",
		canonicalPath: filepath.Join(t.TempDir(), "canonical.ndjson"),
		stampedPath:   filepath.Join(t.TempDir(), "raw.ndjson"),
	})

	session.observeCanonicalEvent(parse.CanonicalEvent{Type: parse.EventResult})
	session.closeStreamDone()
	session.resolveAndMarkDone(1, false)

	if got := session.Status(); got != "completed" {
		t.Fatalf("status = %q, want completed", got)
	}
	outcome := session.Outcome()
	if outcome.Status != StatusCompleted {
		t.Fatalf("outcome status = %q, want %q", outcome.Status, StatusCompleted)
	}
	if !outcome.HasDeliverable {
		t.Fatal("HasDeliverable = false, want true")
	}
	if outcome.ExitCode != 1 {
		t.Fatalf("exit code = %d, want 1", outcome.ExitCode)
	}
}

func TestSpritesSessionWritesHeartbeat(t *testing.T) {
	sessionDir := t.TempDir()
	canonicalPath := filepath.Join(sessionDir, "canonical.ndjson")

	session := newSpritesSession(spritesSessionConfig{
		id:            "session-a",
		canonicalPath: canonicalPath,
		stampedPath:   filepath.Join(sessionDir, "raw.ndjson"),
	})

	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	session.consumeCanonicalLine(marshalCanonical(t, parse.CanonicalEvent{
		Type:      parse.EventAction,
		Message:   "apply patch",
		Timestamp: ts,
	}), session.observeCanonicalEvent)

	data, err := os.ReadFile(filepath.Join(sessionDir, "heartbeat.json"))
	if err != nil {
		t.Fatalf("read heartbeat: %v", err)
	}
	var heartbeat struct {
		Timestamp  time.Time `json:"timestamp"`
		TTLSeconds int       `json:"ttl_seconds"`
	}
	if err := json.Unmarshal(data, &heartbeat); err != nil {
		t.Fatalf("parse heartbeat: %v", err)
	}
	if !heartbeat.Timestamp.Equal(ts) {
		t.Fatalf("heartbeat timestamp = %s, want %s", heartbeat.Timestamp, ts)
	}
	if heartbeat.TTLSeconds != sessionHeartbeatTTLSeconds {
		t.Fatalf("heartbeat ttl = %d, want %d", heartbeat.TTLSeconds, sessionHeartbeatTTLSeconds)
	}
}

func TestSpritesSessionResolveAndMarkDoneFailedWithoutEvents(t *testing.T) {
	session := newSpritesSession(spritesSessionConfig{
		id:            "session-a",
		canonicalPath: filepath.Join(t.TempDir(), "canonical.ndjson"),
		stampedPath:   filepath.Join(t.TempDir(), "raw.ndjson"),
	})

	session.closeStreamDone()
	session.resolveAndMarkDone(1, false)

	if got := session.Status(); got != "failed" {
		t.Fatalf("status = %q, want failed", got)
	}
	outcome := session.Outcome()
	if outcome.Status != StatusFailed {
		t.Fatalf("outcome status = %q, want %q", outcome.Status, StatusFailed)
	}
	if outcome.HasDeliverable {
		t.Fatal("HasDeliverable = true, want false")
	}
	if outcome.Reason != "no events emitted" {
		t.Fatalf("outcome reason = %q, want no events emitted", outcome.Reason)
	}
}

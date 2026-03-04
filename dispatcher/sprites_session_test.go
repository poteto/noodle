package dispatcher

import (
	"path/filepath"
	"testing"

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

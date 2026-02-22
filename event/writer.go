package event

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EventWriter appends events to one session log.
type EventWriter struct {
	runtimeDir string
	sessionID  string
	now        func() time.Time
}

// NewEventWriter constructs a writer for one session.
func NewEventWriter(runtimeDir, sessionID string) (*EventWriter, error) {
	runtimeDir = strings.TrimSpace(runtimeDir)
	sessionID = strings.TrimSpace(sessionID)
	if runtimeDir == "" {
		return nil, fmt.Errorf("runtime directory is required")
	}
	if sessionID == "" {
		return nil, fmt.Errorf("session ID is required")
	}
	if err := os.MkdirAll(sessionDir(runtimeDir, sessionID), 0o755); err != nil {
		return nil, fmt.Errorf("create session directory: %w", err)
	}
	return &EventWriter{
		runtimeDir: runtimeDir,
		sessionID:  sessionID,
		now:        time.Now,
	}, nil
}

// Append writes one event as NDJSON.
func (w *EventWriter) Append(ctx context.Context, event Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if event.Type == "" {
		return fmt.Errorf("event type is required")
	}
	if event.SessionID == "" {
		event.SessionID = w.sessionID
	}
	if event.SessionID != w.sessionID {
		return fmt.Errorf("event session ID %q does not match writer session %q", event.SessionID, w.sessionID)
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = w.now().UTC()
	} else {
		event.Timestamp = event.Timestamp.UTC()
	}

	encoded, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode event: %w", err)
	}

	path := sessionEventsPath(w.runtimeDir, w.sessionID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create events directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open event log: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("append event: %w", err)
	}
	return nil
}

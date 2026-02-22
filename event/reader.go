package event

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const defaultMaxEventLineBytes = 64 << 20

// EventFilter limits event reads by type and time range.
type EventFilter struct {
	Types map[EventType]struct{}
	Since *time.Time
	Until *time.Time
}

func (f EventFilter) match(event Event) bool {
	if len(f.Types) > 0 {
		if _, ok := f.Types[event.Type]; !ok {
			return false
		}
	}
	if f.Since != nil && event.Timestamp.Before(f.Since.UTC()) {
		return false
	}
	if f.Until != nil && event.Timestamp.After(f.Until.UTC()) {
		return false
	}
	return true
}

// EventReader reads session event logs.
type EventReader struct {
	runtimeDir   string
	maxLineBytes int
}

// NewEventReader constructs a reader rooted at a .noodle runtime directory.
func NewEventReader(runtimeDir string) *EventReader {
	return &EventReader{
		runtimeDir:   strings.TrimSpace(runtimeDir),
		maxLineBytes: defaultMaxEventLineBytes,
	}
}

// ReadSession reads events for one session and applies the filter.
func (r *EventReader) ReadSession(sessionID string, filter EventFilter) ([]Event, error) {
	path := sessionEventsPath(r.runtimeDir, strings.TrimSpace(sessionID))
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open event log: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), r.maxLineBytes)

	events := make([]Event, 0, 32)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("parse event line %d: %w", lineNumber, err)
		}
		if filter.match(event) {
			events = append(events, event)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read event log: %w", err)
	}
	return events, nil
}

// TailSession reads the most recent n events for one session after filtering.
func (r *EventReader) TailSession(sessionID string, n int, filter EventFilter) ([]Event, error) {
	events, err := r.ReadSession(sessionID, filter)
	if err != nil {
		return nil, err
	}
	if n <= 0 || len(events) <= n {
		return events, nil
	}
	return events[len(events)-n:], nil
}

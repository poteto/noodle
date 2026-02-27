package event

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// LoopEventType is a lifecycle event emitted by the loop.
type LoopEventType string

const (
	LoopEventStageCompleted     LoopEventType = "stage.completed"
	LoopEventStageFailed        LoopEventType = "stage.failed"
	LoopEventOrderCompleted     LoopEventType = "order.completed"
	LoopEventOrderFailed        LoopEventType = "order.failed"
	LoopEventOrderDropped       LoopEventType = "order.dropped"
	LoopEventOrderRequeued      LoopEventType = "order.requeued"
	LoopEventWorktreeMerged     LoopEventType = "worktree.merged"
	LoopEventMergeConflict      LoopEventType = "merge.conflict"
	LoopEventQualityWritten     LoopEventType = "quality.written"
	LoopEventScheduleCompleted  LoopEventType = "schedule.completed"
	LoopEventRegistryRebuilt    LoopEventType = "registry.rebuilt"
	LoopEventSyncDegraded       LoopEventType = "sync.degraded"
	LoopEventBootstrapCompleted LoopEventType = "bootstrap.completed"
	LoopEventBootstrapExhausted LoopEventType = "bootstrap.exhausted"
)

// LoopEvent is one append-only NDJSON record in loop-events.ndjson.
type LoopEvent struct {
	Seq     uint64          `json:"seq"`
	Type    LoopEventType   `json:"type"`
	At      time.Time       `json:"at"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

const maxLoopEventLines = 200

// LoopEventWriter appends lifecycle events to an NDJSON file.
// It is safe for concurrent use.
type LoopEventWriter struct {
	mu   sync.Mutex
	path string
	seq  uint64
	now  func() time.Time
}

// NewLoopEventWriter creates a writer targeting the given file path.
// It reads the file to determine the next sequence number (continuing
// from the highest existing sequence).
func NewLoopEventWriter(path string) *LoopEventWriter {
	w := &LoopEventWriter{
		path: path,
		now:  time.Now,
	}
	w.seq = w.readHighestSeq()
	return w
}

// Emit appends a lifecycle event. The payload is marshaled to JSON.
// On write failure, a warning is logged to stderr and the error is
// returned — callers should ignore the error (best-effort emission).
func (w *LoopEventWriter) Emit(eventType LoopEventType, payload any) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.seq++

	var raw json.RawMessage
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			fmt.Fprintf(os.Stderr, "loop-event: marshal payload: %v\n", err)
			return err
		}
		raw = b
	}

	ev := LoopEvent{
		Seq:     w.seq,
		Type:    eventType,
		At:      w.now().UTC(),
		Payload: raw,
	}

	data, err := json.Marshal(ev)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loop-event: marshal event: %v\n", err)
		return err
	}

	f, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loop-event: open file: %v\n", err)
		return err
	}
	_, writeErr := f.Write(append(data, '\n'))
	closeErr := f.Close()

	if writeErr != nil {
		fmt.Fprintf(os.Stderr, "loop-event: write: %v\n", writeErr)
		return writeErr
	}
	if closeErr != nil {
		fmt.Fprintf(os.Stderr, "loop-event: close: %v\n", closeErr)
		return closeErr
	}

	w.truncate()
	return nil
}

// Seq returns the current sequence number.
func (w *LoopEventWriter) Seq() uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.seq
}

// readHighestSeq scans the existing file for the highest sequence number.
func (w *LoopEventWriter) readHighestSeq() uint64 {
	f, err := os.Open(w.path)
	if err != nil {
		return 0
	}
	defer f.Close()

	var highest uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev struct {
			Seq uint64 `json:"seq"`
		}
		if json.Unmarshal([]byte(line), &ev) == nil && ev.Seq > highest {
			highest = ev.Seq
		}
	}
	return highest
}

// truncate keeps only the last maxLoopEventLines lines.
func (w *LoopEventWriter) truncate() {
	f, err := os.Open(w.path)
	if err != nil {
		return
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	_ = f.Close()

	if len(lines) <= maxLoopEventLines {
		return
	}

	lines = lines[len(lines)-maxLoopEventLines:]
	var buf strings.Builder
	for _, line := range lines {
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	_ = os.WriteFile(w.path, []byte(buf.String()), 0o644)
}

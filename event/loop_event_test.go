package event

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestLoopEventWriter_Emit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loop-events.ndjson")
	w := NewLoopEventWriter(path)

	type testPayload struct {
		OrderID string `json:"order_id"`
	}

	if err := w.Emit(LoopEventStageCompleted, testPayload{OrderID: "o1"}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if err := w.Emit(LoopEventStageFailed, testPayload{OrderID: "o2"}); err != nil {
		t.Fatalf("emit: %v", err)
	}

	events := readAllLoopEvents(t, path)
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}

	if events[0].Seq != 1 {
		t.Errorf("event[0].Seq = %d, want 1", events[0].Seq)
	}
	if events[1].Seq != 2 {
		t.Errorf("event[1].Seq = %d, want 2", events[1].Seq)
	}
	if events[0].Type != LoopEventStageCompleted {
		t.Errorf("event[0].Type = %q, want %q", events[0].Type, LoopEventStageCompleted)
	}
	if events[1].Type != LoopEventStageFailed {
		t.Errorf("event[1].Type = %q, want %q", events[1].Type, LoopEventStageFailed)
	}
	if events[0].At.IsZero() {
		t.Error("event[0].At is zero")
	}

	var p testPayload
	if err := json.Unmarshal(events[0].Payload, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if p.OrderID != "o1" {
		t.Errorf("payload.OrderID = %q, want %q", p.OrderID, "o1")
	}
}

func TestLoopEventWriter_NilPayload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loop-events.ndjson")
	w := NewLoopEventWriter(path)

	if err := w.Emit(LoopEventBootstrapCompleted, nil); err != nil {
		t.Fatalf("emit: %v", err)
	}

	events := readAllLoopEvents(t, path)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Payload != nil {
		t.Errorf("payload = %s, want nil", string(events[0].Payload))
	}
}

func TestLoopEventWriter_MonotonicSequences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loop-events.ndjson")
	w := NewLoopEventWriter(path)

	for i := 0; i < 10; i++ {
		if err := w.Emit(LoopEventRegistryRebuilt, nil); err != nil {
			t.Fatalf("emit %d: %v", i, err)
		}
	}

	events := readAllLoopEvents(t, path)
	for i, ev := range events {
		want := uint64(i + 1)
		if ev.Seq != want {
			t.Errorf("event[%d].Seq = %d, want %d", i, ev.Seq, want)
		}
	}
}

func TestLoopEventWriter_ConcurrentSafe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loop-events.ndjson")
	w := NewLoopEventWriter(path)

	const goroutines = 10
	const perGoroutine = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				_ = w.Emit(LoopEventStageCompleted, nil)
			}
		}()
	}
	wg.Wait()

	events := readAllLoopEvents(t, path)

	// With truncation to 200, we might have fewer lines if total > 200.
	// But 10*20 = 200, which is exactly the limit, so all should be present.
	if len(events) != goroutines*perGoroutine {
		t.Fatalf("got %d events, want %d", len(events), goroutines*perGoroutine)
	}

	// All sequences should be unique and monotonically increasing.
	seen := make(map[uint64]bool)
	for _, ev := range events {
		if seen[ev.Seq] {
			t.Errorf("duplicate sequence %d", ev.Seq)
		}
		seen[ev.Seq] = true
	}
	if len(seen) != goroutines*perGoroutine {
		t.Errorf("got %d unique sequences, want %d", len(seen), goroutines*perGoroutine)
	}
}

func TestLoopEventWriter_Truncation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loop-events.ndjson")
	w := NewLoopEventWriter(path)

	// Write more than maxLoopEventLines events.
	total := maxLoopEventLines + 50
	for i := 0; i < total; i++ {
		if err := w.Emit(LoopEventStageCompleted, nil); err != nil {
			t.Fatalf("emit %d: %v", i, err)
		}
	}

	events := readAllLoopEvents(t, path)
	if len(events) != maxLoopEventLines {
		t.Fatalf("got %d events, want %d (truncated)", len(events), maxLoopEventLines)
	}

	// First event should be seq 51 (we kept the last 200 of 250).
	if events[0].Seq != uint64(total-maxLoopEventLines+1) {
		t.Errorf("first event seq = %d, want %d", events[0].Seq, total-maxLoopEventLines+1)
	}
	// Last event should be the most recent.
	if events[len(events)-1].Seq != uint64(total) {
		t.Errorf("last event seq = %d, want %d", events[len(events)-1].Seq, total)
	}
}

func TestLoopEventWriter_ContinuesSeqFromExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loop-events.ndjson")

	// Write some events with a first writer.
	w1 := NewLoopEventWriter(path)
	for i := 0; i < 5; i++ {
		_ = w1.Emit(LoopEventStageCompleted, nil)
	}

	// Create a new writer — should continue from seq 5.
	w2 := NewLoopEventWriter(path)
	_ = w2.Emit(LoopEventStageFailed, nil)

	events := readAllLoopEvents(t, path)
	if len(events) != 6 {
		t.Fatalf("got %d events, want 6", len(events))
	}
	if events[5].Seq != 6 {
		t.Errorf("event[5].Seq = %d, want 6", events[5].Seq)
	}
}

func TestLoopEventWriter_WriteFailureLogs(t *testing.T) {
	// Point writer at a directory (not a file) to force a write failure.
	dir := t.TempDir()
	w := NewLoopEventWriter(dir)

	err := w.Emit(LoopEventStageCompleted, nil)
	if err == nil {
		t.Error("expected error for directory path, got nil")
	}
	// Writer did not panic — that's the key property.
}

func TestLoopEventWriter_EmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loop-events.ndjson")
	// No file exists yet.
	w := NewLoopEventWriter(path)
	if w.Seq() != 0 {
		t.Errorf("initial seq = %d, want 0", w.Seq())
	}
}

func TestLoopEventWriter_TimestampUTC(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loop-events.ndjson")
	w := NewLoopEventWriter(path)
	fixed := time.Date(2026, 2, 27, 12, 0, 0, 0, time.FixedZone("PST", -8*3600))
	w.now = func() time.Time { return fixed }

	_ = w.Emit(LoopEventStageCompleted, nil)

	events := readAllLoopEvents(t, path)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].At.Location() != time.UTC {
		t.Errorf("timestamp location = %v, want UTC", events[0].At.Location())
	}
}

// readAllLoopEvents parses every line from the NDJSON file.
func readAllLoopEvents(t *testing.T, path string) []LoopEvent {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	var events []LoopEvent
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var ev LoopEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			t.Fatalf("unmarshal: %v (line: %s)", err, scanner.Text())
		}
		events = append(events, ev)
	}
	return events
}

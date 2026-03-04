package dispatcher

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/parse"
)

func TestProcessSessionClosesEventsAfterDone(t *testing.T) {
	cmd := exec.Command("echo", "done")
	process, err := StartProcess(cmd)
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}

	dir := t.TempDir()
	session := newProcessSession(processSessionConfig{
		id:            "session-a",
		process:       process,
		canonicalPath: filepath.Join(dir, "canonical.ndjson"),
		stampedPath:   filepath.Join(dir, "raw.ndjson"),
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session.start(ctx)

	select {
	case <-session.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("session did not signal done")
	}

	timeout := time.After(5 * time.Second)
	for {
		select {
		case _, ok := <-session.Events():
			if !ok {
				return
			}
		case <-timeout:
			t.Fatal("events channel did not close")
		}
	}
}

func TestProcessSessionWritesEventLog(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	writer, err := event.NewEventWriter(runtimeDir, "session-a")
	if err != nil {
		t.Fatalf("new event writer: %v", err)
	}

	cmd := exec.Command("echo", "noop")
	process, err := StartProcess(cmd)
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	<-process.Done()

	session := newProcessSession(processSessionConfig{
		id:            "session-a",
		process:       process,
		eventWriter:   writer,
		canonicalPath: filepath.Join(t.TempDir(), "canonical.ndjson"),
		stampedPath:   filepath.Join(t.TempDir(), "raw.ndjson"),
	})

	session.consumeCanonicalLine(marshalCanonical(t, parse.CanonicalEvent{
		Type:      parse.EventAction,
		Message:   "apply patch",
		Timestamp: time.Date(2026, 2, 27, 20, 0, 0, 0, time.UTC),
	}), nil)
	session.consumeCanonicalLine(marshalCanonical(t, parse.CanonicalEvent{
		Type:      parse.EventResult,
		CostUSD:   0.12,
		TokensIn:  100,
		TokensOut: 50,
		Timestamp: time.Date(2026, 2, 27, 20, 0, 1, 0, time.UTC),
	}), nil)

	reader := event.NewEventReader(runtimeDir)
	records, err := reader.ReadSession("session-a", event.EventFilter{})
	if err != nil {
		t.Fatalf("read event log: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("event count = %d, want 2", len(records))
	}
	if records[0].Type != event.EventAction {
		t.Fatalf("first event type = %q", records[0].Type)
	}
	if records[1].Type != event.EventCost {
		t.Fatalf("second event type = %q", records[1].Type)
	}
}

func TestProcessSessionWritesHeartbeat(t *testing.T) {
	sessionDir := t.TempDir()
	canonicalPath := filepath.Join(sessionDir, "canonical.ndjson")

	cmd := exec.Command("echo", "noop")
	process, err := StartProcess(cmd)
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	<-process.Done()

	session := newProcessSession(processSessionConfig{
		id:            "session-a",
		process:       process,
		canonicalPath: canonicalPath,
		stampedPath:   filepath.Join(sessionDir, "raw.ndjson"),
	})

	ts := time.Date(2026, 2, 27, 6, 0, 0, 0, time.UTC)
	session.consumeCanonicalLine(marshalCanonical(t, parse.CanonicalEvent{
		Type:      parse.EventAction,
		Message:   "check status",
		Timestamp: ts,
	}), session.processHook)

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

func TestProcessSessionKillMarksKilled(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	process, err := StartProcess(cmd)
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}

	session := newProcessSession(processSessionConfig{
		id:            "session-a",
		process:       process,
		canonicalPath: filepath.Join(t.TempDir(), "canonical.ndjson"),
		stampedPath:   filepath.Join(t.TempDir(), "raw.ndjson"),
	})

	if err := session.ForceKill(); err != nil {
		t.Fatalf("Kill: %v", err)
	}

	select {
	case <-session.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("session did not signal done after kill")
	}

	if session.Status() != "killed" {
		t.Fatalf("status = %q, want killed", session.Status())
	}
}

func TestProcessSessionResolveAndMarkDoneCompletedFromResult(t *testing.T) {
	cmd := exec.Command("echo", "noop")
	process, err := StartProcess(cmd)
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	<-process.Done()

	session := newProcessSession(processSessionConfig{
		id:            "session-a",
		process:       process,
		canonicalPath: filepath.Join(t.TempDir(), "canonical.ndjson"),
		stampedPath:   filepath.Join(t.TempDir(), "raw.ndjson"),
	})

	session.processHook(parse.CanonicalEvent{Type: parse.EventResult, Timestamp: nowUTC()})
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

func TestProcessSessionResolveAndMarkDoneCancellationWithoutCompletion(t *testing.T) {
	cmd := exec.Command("echo", "noop")
	process, err := StartProcess(cmd)
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	<-process.Done()

	session := newProcessSession(processSessionConfig{
		id:            "session-a",
		process:       process,
		canonicalPath: filepath.Join(t.TempDir(), "canonical.ndjson"),
		stampedPath:   filepath.Join(t.TempDir(), "raw.ndjson"),
	})

	session.closeStreamDone()
	session.resolveAndMarkDone(1, true)

	if got := session.Status(); got != "cancelled" {
		t.Fatalf("status = %q, want cancelled", got)
	}
	outcome := session.Outcome()
	if outcome.Status != StatusCancelled {
		t.Fatalf("outcome status = %q, want %q", outcome.Status, StatusCancelled)
	}
}

func TestProcessSessionResolveAndMarkDoneCompletionWinsOverCancellation(t *testing.T) {
	cmd := exec.Command("echo", "noop")
	process, err := StartProcess(cmd)
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	<-process.Done()

	session := newProcessSession(processSessionConfig{
		id:            "session-a",
		process:       process,
		canonicalPath: filepath.Join(t.TempDir(), "canonical.ndjson"),
		stampedPath:   filepath.Join(t.TempDir(), "raw.ndjson"),
	})

	session.processHook(parse.CanonicalEvent{Type: parse.EventResult, Timestamp: nowUTC()})
	session.closeStreamDone()
	session.resolveAndMarkDone(1, true)

	if got := session.Status(); got != "completed" {
		t.Fatalf("status = %q, want completed", got)
	}
	outcome := session.Outcome()
	if outcome.Status != StatusCompleted {
		t.Fatalf("outcome status = %q, want %q", outcome.Status, StatusCompleted)
	}
}

func TestProcessSessionResolveAndMarkDoneActionWithoutCompletionFails(t *testing.T) {
	cmd := exec.Command("echo", "noop")
	process, err := StartProcess(cmd)
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	<-process.Done()

	session := newProcessSession(processSessionConfig{
		id:            "session-a",
		process:       process,
		canonicalPath: filepath.Join(t.TempDir(), "canonical.ndjson"),
		stampedPath:   filepath.Join(t.TempDir(), "raw.ndjson"),
	})

	session.processHook(parse.CanonicalEvent{Type: parse.EventAction, Timestamp: nowUTC()})
	session.closeStreamDone()
	session.resolveAndMarkDone(1, false)

	if got := session.Status(); got != "failed" {
		t.Fatalf("status = %q, want failed", got)
	}
	outcome := session.Outcome()
	if outcome.Status != StatusFailed {
		t.Fatalf("outcome status = %q, want %q", outcome.Status, StatusFailed)
	}
	if outcome.Reason != "no turn completed" {
		t.Fatalf("outcome reason = %q, want no turn completed", outcome.Reason)
	}
}

func TestProcessSessionResolveAndMarkDoneSignalExitKilled(t *testing.T) {
	cmd := exec.Command("echo", "noop")
	process, err := StartProcess(cmd)
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	<-process.Done()

	session := newProcessSession(processSessionConfig{
		id:            "session-a",
		process:       process,
		canonicalPath: filepath.Join(t.TempDir(), "canonical.ndjson"),
		stampedPath:   filepath.Join(t.TempDir(), "raw.ndjson"),
	})

	session.closeStreamDone()
	session.resolveAndMarkDone(-1, false)

	if got := session.Status(); got != "killed" {
		t.Fatalf("status = %q, want killed", got)
	}
	if got := session.Outcome().Status; got != StatusKilled {
		t.Fatalf("outcome status = %q, want %q", got, StatusKilled)
	}
}

func TestProcessSessionCostAccumulation(t *testing.T) {
	cmd := exec.Command("echo", "noop")
	process, err := StartProcess(cmd)
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	<-process.Done()

	session := newProcessSession(processSessionConfig{
		id:            "session-a",
		process:       process,
		canonicalPath: filepath.Join(t.TempDir(), "canonical.ndjson"),
		stampedPath:   filepath.Join(t.TempDir(), "raw.ndjson"),
	})

	session.consumeCanonicalLine(marshalCanonical(t, parse.CanonicalEvent{
		Type:      parse.EventResult,
		CostUSD:   0.10,
		Timestamp: time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC),
	}), nil)
	session.consumeCanonicalLine(marshalCanonical(t, parse.CanonicalEvent{
		Type:      parse.EventResult,
		CostUSD:   0.25,
		Timestamp: time.Date(2026, 2, 27, 0, 0, 1, 0, time.UTC),
	}), nil)

	if got := session.TotalCost(); got != 0.35 {
		t.Fatalf("TotalCost = %f, want 0.35", got)
	}
}

func TestProcessSessionEmitsPromptOnInit(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	writer, err := event.NewEventWriter(runtimeDir, "session-a")
	if err != nil {
		t.Fatalf("new event writer: %v", err)
	}

	prompt := "Build the authentication module"

	cmd := exec.Command("echo", "noop")
	process, err := StartProcess(cmd)
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	<-process.Done()

	session := newProcessSession(processSessionConfig{
		id:            "session-a",
		process:       process,
		eventWriter:   writer,
		canonicalPath: filepath.Join(t.TempDir(), "canonical.ndjson"),
		stampedPath:   filepath.Join(t.TempDir(), "raw.ndjson"),
		prompt:        prompt,
	})

	ts := time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC)
	session.consumeCanonicalLine(marshalCanonical(t, parse.CanonicalEvent{
		Type:      parse.EventInit,
		Message:   "session started",
		Timestamp: ts,
	}), nil)

	reader := event.NewEventReader(runtimeDir)
	records, err := reader.ReadSession("session-a", event.EventFilter{})
	if err != nil {
		t.Fatalf("read event log: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("event count = %d, want 2", len(records))
	}
	if records[0].Type != event.EventSpawned {
		t.Fatalf("first event type = %q", records[0].Type)
	}
	if records[1].Type != event.EventAction {
		t.Fatalf("second event type = %q", records[1].Type)
	}

	var payload struct {
		Tool    string `json:"tool"`
		Action  string `json:"action"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(records[1].Payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Tool != "prompt" {
		t.Fatalf("tool = %q, want prompt", payload.Tool)
	}
	if payload.Action != "prompt_injected" {
		t.Fatalf("action = %q, want prompt_injected", payload.Action)
	}
	if payload.Message != prompt {
		t.Fatalf("message = %q, want %q", payload.Message, prompt)
	}
}

func TestProcessSessionDoesNotEmitSpawnedOnSubsequentInits(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	writer, err := event.NewEventWriter(runtimeDir, "session-a")
	if err != nil {
		t.Fatalf("new event writer: %v", err)
	}

	cmd := exec.Command("echo", "noop")
	process, err := StartProcess(cmd)
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	<-process.Done()

	session := newProcessSession(processSessionConfig{
		id:            "session-a",
		process:       process,
		eventWriter:   writer,
		canonicalPath: filepath.Join(t.TempDir(), "canonical.ndjson"),
		stampedPath:   filepath.Join(t.TempDir(), "raw.ndjson"),
		prompt:        "initial prompt",
	})

	ts1 := time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC)
	session.consumeCanonicalLine(marshalCanonical(t, parse.CanonicalEvent{
		Type:      parse.EventInit,
		Message:   "session started",
		Timestamp: ts1,
	}), nil)

	ts2 := time.Date(2026, 2, 27, 0, 0, 5, 0, time.UTC)
	session.consumeCanonicalLine(marshalCanonical(t, parse.CanonicalEvent{
		Type:      parse.EventInit,
		Message:   "new turn started",
		Timestamp: ts2,
	}), nil)

	reader := event.NewEventReader(runtimeDir)
	records, err := reader.ReadSession("session-a", event.EventFilter{})
	if err != nil {
		t.Fatalf("read event log: %v", err)
	}

	spawnedCount := 0
	for _, record := range records {
		if record.Type == event.EventSpawned {
			spawnedCount++
		}
	}
	if spawnedCount != 1 {
		t.Fatalf("spawned count = %d, want 1", spawnedCount)
	}
}

func TestProcessSessionDroppedEventSummary(t *testing.T) {
	cmd := exec.Command("echo", "noop")
	process, err := StartProcess(cmd)
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	<-process.Done()

	session := newProcessSession(processSessionConfig{
		id:            "session-a",
		process:       process,
		canonicalPath: filepath.Join(t.TempDir(), "canonical.ndjson"),
		stampedPath:   filepath.Join(t.TempDir(), "raw.ndjson"),
	})

	for i := 0; i < cap(session.events); i++ {
		session.events <- SessionEvent{Type: "action", Message: "seed"}
	}
	session.publish(SessionEvent{Type: "action", Message: "newest"})

	session.markDone("completed")
	session.closeEventsWhenDone()

	found := false
	for ev := range session.Events() {
		if ev.Type == "warning" && strings.Contains(ev.Message, "dropped") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected dropped-events warning")
	}
}

func marshalCanonical(t *testing.T, ce parse.CanonicalEvent) []byte {
	t.Helper()
	data, err := json.Marshal(ce)
	if err != nil {
		t.Fatalf("marshal canonical: %v", err)
	}
	return data
}

package loop

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/mise"
)

// readNDJSON reads loop-events.ndjson and returns parsed events.
func readNDJSON(t *testing.T, path string) []event.LoopEvent {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read events file: %v", err)
	}
	var events []event.LoopEvent
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev event.LoopEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("unmarshal event line: %v", err)
		}
		events = append(events, ev)
	}
	return events
}

// findEvents returns all events of the given type.
func findEvents(events []event.LoopEvent, eventType event.LoopEventType) []event.LoopEvent {
	var result []event.LoopEvent
	for _, ev := range events {
		if ev.Type == eventType {
			result = append(result, ev)
		}
	}
	return result
}

func TestEventIntegrationStageCompleted(t *testing.T) {
	// Drive a cook through completion, verify stage.completed event is emitted
	// and appears in the mise brief's recent_events.
	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     "ev-1",
				Title:  "event test",
				Status: OrderStatusActive,
				Stages: []Stage{
					{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
					{TaskKey: "review", Skill: "review", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
				},
			},
		},
	}

	env := newIntegrationEnv(t, orders)
	eventsPath := filepath.Join(env.runtimeDir, "loop-events.ndjson")

	// Cycle 1: dispatch stage 0.
	if err := env.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}

	// Complete the session.
	env.completeSessions("completed")

	// Cycle 2: handle completion, advance stage.
	if err := env.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}

	// Read events file: should contain stage.completed.
	events := readNDJSON(t, eventsPath)
	completed := findEvents(events, event.LoopEventStageCompleted)
	if len(completed) == 0 {
		t.Fatal("no stage.completed events found")
	}

	// Verify payload.
	var payload StageCompletedPayload
	if err := json.Unmarshal(completed[0].Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.OrderID != "ev-1" {
		t.Fatalf("payload order_id = %q, want ev-1", payload.OrderID)
	}
	if payload.TaskKey != "execute" {
		t.Fatalf("payload task_key = %q, want execute", payload.TaskKey)
	}

	// Build a mise brief and verify recent_events contains the event.
	builder := mise.NewBuilder(env.projectDir, env.loop.config)

	brief, _, _, err := builder.Build(context.Background(), mise.ActiveSummary{}, nil)
	if err != nil {
		t.Fatalf("build brief: %v", err)
	}
	if len(brief.RecentEvents) == 0 {
		t.Fatal("brief.RecentEvents is empty, expected stage.completed")
	}
	found := false
	for _, re := range brief.RecentEvents {
		if re.Type == "stage.completed" {
			found = true
		}
	}
	if !found {
		t.Fatal("stage.completed not found in brief.RecentEvents")
	}
}

func TestEventIntegrationFailurePath(t *testing.T) {
	// Cook fails → stage.failed + order.failed events appear.
	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     "fail-1",
				Title:  "failure test",
				Status: OrderStatusActive,
				Stages: []Stage{
					{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
				},
			},
		},
	}

	env := newIntegrationEnv(t, orders)
	eventsPath := filepath.Join(env.runtimeDir, "loop-events.ndjson")

	// Cycle 1: dispatch.
	if err := env.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}

	// Complete with failure — first failure is terminal.
	env.completeSessions("failed")
	if err := env.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}

	events := readNDJSON(t, eventsPath)
	stageFailed := findEvents(events, event.LoopEventStageFailed)
	if len(stageFailed) == 0 {
		t.Fatal("no stage.failed events found")
	}
	orderFailed := findEvents(events, event.LoopEventOrderFailed)
	if len(orderFailed) == 0 {
		t.Fatal("no order.failed events found after terminal failure")
	}
}

func TestEventIntegrationRequeueEmitsEvent(t *testing.T) {
	// Control command: requeue → order.requeued event appears.
	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     "rq-1",
				Title:  "requeue test",
				Status: OrderStatusActive,
				Stages: []Stage{
					{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusFailed},
				},
			},
		},
	}

	env := newIntegrationEnv(t, orders)
	eventsPath := filepath.Join(env.runtimeDir, "loop-events.ndjson")
	l := env.loop

	// Execute requeue control command.
	if err := l.controlRequeue("rq-1"); err != nil {
		t.Fatalf("controlRequeue: %v", err)
	}

	events := readNDJSON(t, eventsPath)
	requeued := findEvents(events, event.LoopEventOrderRequeued)
	if len(requeued) == 0 {
		t.Fatal("no order.requeued events found")
	}
	var payload OrderRequeuedPayload
	if err := json.Unmarshal(requeued[0].Payload, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.OrderID != "rq-1" {
		t.Fatalf("payload order_id = %q, want rq-1", payload.OrderID)
	}
}

func TestEventIntegrationWatermarkAdvances(t *testing.T) {
	// After schedule.completed, only newer events appear in the brief.
	eventsPath := filepath.Join(t.TempDir(), ".noodle", "loop-events.ndjson")
	if err := os.MkdirAll(filepath.Dir(eventsPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	w := event.NewLoopEventWriter(eventsPath)

	// Emit events before schedule.completed.
	_ = w.Emit(event.LoopEventStageCompleted, StageCompletedPayload{OrderID: "old-1", TaskKey: "execute"})
	_ = w.Emit(event.LoopEventScheduleCompleted, ScheduleCompletedPayload{SessionID: "s1"})
	// Emit events after schedule.completed.
	_ = w.Emit(event.LoopEventStageCompleted, StageCompletedPayload{OrderID: "new-1", TaskKey: "review"})
	_ = w.Emit(event.LoopEventOrderCompleted, OrderCompletedPayload{OrderID: "new-1"})

	// Build brief from the same runtimeDir.
	projectDir := filepath.Dir(filepath.Dir(eventsPath))
	builder := mise.NewBuilder(projectDir, testDefaultConfig())

	brief, _, _, err := builder.Build(context.Background(), mise.ActiveSummary{}, nil)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	// Only events after the watermark (seq 2) should appear.
	if len(brief.RecentEvents) != 2 {
		t.Fatalf("recent_events = %d, want 2", len(brief.RecentEvents))
	}
	for _, re := range brief.RecentEvents {
		if re.Seq <= 2 {
			t.Fatalf("event seq %d should be after watermark 2", re.Seq)
		}
	}
}

func TestEventIntegrationExternalEvent(t *testing.T) {
	// External event emitted via EventWriter appears alongside internal events.
	eventsPath := filepath.Join(t.TempDir(), ".noodle", "loop-events.ndjson")
	if err := os.MkdirAll(filepath.Dir(eventsPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	w := event.NewLoopEventWriter(eventsPath)

	// Emit an internal event.
	_ = w.Emit(event.LoopEventStageCompleted, StageCompletedPayload{OrderID: "a", TaskKey: "execute"})
	// Emit an external event (arbitrary type).
	_ = w.Emit("ci.failed", map[string]string{"branch": "main", "reason": "lint error"})

	projectDir := filepath.Dir(filepath.Dir(eventsPath))
	builder := mise.NewBuilder(projectDir, testDefaultConfig())

	brief, _, _, err := builder.Build(context.Background(), mise.ActiveSummary{}, nil)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if len(brief.RecentEvents) != 2 {
		t.Fatalf("recent_events = %d, want 2", len(brief.RecentEvents))
	}

	// Verify the external event is present.
	found := false
	for _, re := range brief.RecentEvents {
		if re.Type == "ci.failed" {
			found = true
		}
	}
	if !found {
		t.Fatal("external event ci.failed not found in brief.RecentEvents")
	}
}

func testDefaultConfig() config.Config {
	cfg := config.DefaultConfig()
	return cfg
}

package event

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestTicketMaterializerLifecycleAndStale(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	now := time.Date(2026, 2, 22, 20, 0, 0, 0, time.UTC)

	writeTicketEvents(t, runtimeDir, "cook-active", []Event{
		ticketEvent(t, "cook-active", EventTicketClaim, now.Add(-10*time.Minute), TicketEventPayload{
			Target:     "42",
			TargetType: TicketTargetBacklogItem,
			Files:      []string{"src/auth/token.go"},
		}),
		ticketEvent(t, "cook-active", EventTicketProgress, now.Add(-4*time.Minute), TicketEventPayload{
			Target:     "42",
			TargetType: TicketTargetBacklogItem,
			Summary:    "still implementing",
		}),
	})

	writeTicketEvents(t, runtimeDir, "cook-blocked", []Event{
		ticketEvent(t, "cook-blocked", EventTicketClaim, now.Add(-12*time.Minute), TicketEventPayload{
			Target:     "phase-03",
			TargetType: TicketTargetPlanPhase,
		}),
		ticketEvent(t, "cook-blocked", EventTicketBlocked, now.Add(-2*time.Minute), TicketEventPayload{
			Target:     "phase-03",
			TargetType: TicketTargetPlanPhase,
			BlockedBy:  "42",
			Reason:     "waiting for backlog item 42",
		}),
	})

	writeTicketEvents(t, runtimeDir, "cook-stale", []Event{
		ticketEvent(t, "cook-stale", EventTicketClaim, now.Add(-45*time.Minute), TicketEventPayload{
			Target:     "src/legacy/file.go",
			TargetType: TicketTargetFile,
		}),
	})

	materializer := NewTicketMaterializer(runtimeDir)
	materializer.now = func() time.Time { return now }
	materializer.staleTimeout = 30 * time.Minute

	tickets, err := materializer.Materialize(context.Background(), nil)
	if err != nil {
		t.Fatalf("materialize tickets: %v", err)
	}
	if len(tickets) != 3 {
		t.Fatalf("ticket count = %d, want 3", len(tickets))
	}

	statusByTarget := map[string]TicketStatus{}
	for _, ticket := range tickets {
		statusByTarget[ticket.Target] = ticket.Status
	}
	if got := statusByTarget["42"]; got != TicketStatusActive {
		t.Fatalf("ticket 42 status = %q, want %q", got, TicketStatusActive)
	}
	if got := statusByTarget["phase-03"]; got != TicketStatusBlocked {
		t.Fatalf("ticket phase-03 status = %q, want %q", got, TicketStatusBlocked)
	}
	if got := statusByTarget["src/legacy/file.go"]; got != TicketStatusStale {
		t.Fatalf("stale ticket status = %q, want %q", got, TicketStatusStale)
	}

	active := ActiveTickets(tickets)
	if len(active) != 2 {
		t.Fatalf("active tickets count = %d, want 2", len(active))
	}

	materializedPath := ticketsPath(runtimeDir)
	data, err := os.ReadFile(materializedPath)
	if err != nil {
		t.Fatalf("read tickets.json: %v", err)
	}
	var fromDisk []Ticket
	if err := json.Unmarshal(data, &fromDisk); err != nil {
		t.Fatalf("decode tickets.json: %v", err)
	}
	if !reflect.DeepEqual(tickets, fromDisk) {
		t.Fatalf("tickets.json content mismatch\ndisk: %#v\nmem:  %#v", fromDisk, tickets)
	}
}

func TestTicketMaterializerSkipsInactiveSessionTickets(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	now := time.Date(2026, 2, 22, 20, 30, 0, 0, time.UTC)

	writeTicketEvents(t, runtimeDir, "cook-alive", []Event{
		ticketEvent(t, "cook-alive", EventTicketClaim, now.Add(-8*time.Minute), TicketEventPayload{
			Target:     "99",
			TargetType: TicketTargetBacklogItem,
		}),
	})
	writeTicketEvents(t, runtimeDir, "cook-killed", []Event{
		ticketEvent(t, "cook-killed", EventTicketClaim, now.Add(-7*time.Minute), TicketEventPayload{
			Target:     "100",
			TargetType: TicketTargetBacklogItem,
		}),
	})

	materializer := NewTicketMaterializer(runtimeDir)
	materializer.now = func() time.Time { return now }

	// Simulate monitor view after cook-killed exits: only cook-alive is active.
	tickets, err := materializer.Materialize(context.Background(), []string{"cook-alive"})
	if err != nil {
		t.Fatalf("materialize tickets: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("ticket count = %d, want 1", len(tickets))
	}
	if tickets[0].Target != "99" {
		t.Fatalf("remaining target = %q, want 99", tickets[0].Target)
	}
}

func TestTicketMaterializerResolvesDuplicateClaimsByLatestEvent(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	now := time.Date(2026, 2, 22, 21, 0, 0, 0, time.UTC)

	writeTicketEvents(t, runtimeDir, "cook-1", []Event{
		ticketEvent(t, "cook-1", EventTicketClaim, now.Add(-2*time.Minute), TicketEventPayload{
			Target:     "42",
			TargetType: TicketTargetBacklogItem,
		}),
	})
	writeTicketEvents(t, runtimeDir, "cook-2", []Event{
		ticketEvent(t, "cook-2", EventTicketClaim, now.Add(-1*time.Minute), TicketEventPayload{
			Target:     "42",
			TargetType: TicketTargetBacklogItem,
		}),
	})

	materializer := NewTicketMaterializer(runtimeDir)
	materializer.now = func() time.Time { return now }

	tickets, err := materializer.Materialize(context.Background(), []string{"cook-1", "cook-2"})
	if err != nil {
		t.Fatalf("materialize tickets: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("ticket count = %d, want 1", len(tickets))
	}
	if tickets[0].CookID != "cook-2" {
		t.Fatalf("duplicate claim winner = %q, want cook-2", tickets[0].CookID)
	}
	if tickets[0].Status != TicketStatusActive {
		t.Fatalf("ticket status = %q, want %q", tickets[0].Status, TicketStatusActive)
	}
}

func TestTicketMaterializerInfersTargetTypeOnNonClaimEvents(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	now := time.Date(2026, 2, 22, 21, 30, 0, 0, time.UTC)

	writeTicketEvents(t, runtimeDir, "cook-1", []Event{
		ticketEvent(t, "cook-1", EventTicketClaim, now.Add(-5*time.Minute), TicketEventPayload{
			Target:     "42",
			TargetType: TicketTargetBacklogItem,
		}),
		ticketEvent(t, "cook-1", EventTicketProgress, now.Add(-4*time.Minute), TicketEventPayload{
			Target: "42",
		}),
		ticketEvent(t, "cook-1", EventTicketDone, now.Add(-3*time.Minute), TicketEventPayload{
			Target: "42",
		}),
	})

	materializer := NewTicketMaterializer(runtimeDir)
	materializer.now = func() time.Time { return now }

	tickets, err := materializer.Materialize(context.Background(), []string{"cook-1"})
	if err != nil {
		t.Fatalf("materialize tickets: %v", err)
	}
	if len(tickets) != 0 {
		t.Fatalf("ticket count = %d, want 0", len(tickets))
	}
}

func TestTicketMaterializerIgnoresReleaseFromPreviousClaimant(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	now := time.Date(2026, 2, 22, 22, 0, 0, 0, time.UTC)

	writeTicketEvents(t, runtimeDir, "cook-1", []Event{
		ticketEvent(t, "cook-1", EventTicketClaim, now.Add(-3*time.Minute), TicketEventPayload{
			Target:     "42",
			TargetType: TicketTargetBacklogItem,
		}),
	})
	writeTicketEvents(t, runtimeDir, "cook-2", []Event{
		ticketEvent(t, "cook-2", EventTicketClaim, now.Add(-2*time.Minute), TicketEventPayload{
			Target:     "42",
			TargetType: TicketTargetBacklogItem,
		}),
	})
	writeTicketEvents(t, runtimeDir, "cook-1", []Event{
		ticketEvent(t, "cook-1", EventTicketRelease, now.Add(-1*time.Minute), TicketEventPayload{
			Target: "42",
			Reason: "old claim cleanup",
		}),
	})

	materializer := NewTicketMaterializer(runtimeDir)
	materializer.now = func() time.Time { return now }

	tickets, err := materializer.Materialize(context.Background(), []string{"cook-1", "cook-2"})
	if err != nil {
		t.Fatalf("materialize tickets: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("ticket count = %d, want 1", len(tickets))
	}
	if tickets[0].CookID != "cook-2" {
		t.Fatalf("remaining claimant = %q, want cook-2", tickets[0].CookID)
	}
}

func TestTicketMaterializerPreservesEqualTimestampEventOrder(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	now := time.Date(2026, 2, 22, 22, 30, 0, 0, time.UTC)

	shared := now.Add(-2 * time.Minute)
	writeTicketEvents(t, runtimeDir, "cook-1", []Event{
		ticketEvent(t, "cook-1", EventTicketClaim, now.Add(-3*time.Minute), TicketEventPayload{
			Target:     "42",
			TargetType: TicketTargetBacklogItem,
		}),
		// Same timestamp, append order matters.
		ticketEvent(t, "cook-1", EventTicketProgress, shared, TicketEventPayload{
			Target: "42",
		}),
		ticketEvent(t, "cook-1", EventTicketBlocked, shared, TicketEventPayload{
			Target:    "42",
			BlockedBy: "99",
			Reason:    "waiting",
		}),
	})

	materializer := NewTicketMaterializer(runtimeDir)
	materializer.now = func() time.Time { return now }

	tickets, err := materializer.Materialize(context.Background(), []string{"cook-1"})
	if err != nil {
		t.Fatalf("materialize tickets: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("ticket count = %d, want 1", len(tickets))
	}
	if tickets[0].Status != TicketStatusBlocked {
		t.Fatalf("ticket status = %q, want %q", tickets[0].Status, TicketStatusBlocked)
	}
}

func writeTicketEvents(t *testing.T, runtimeDir, sessionID string, events []Event) {
	t.Helper()
	writer, err := NewEventWriter(runtimeDir, sessionID)
	if err != nil {
		t.Fatalf("new event writer for %s: %v", sessionID, err)
	}
	for _, event := range events {
		if err := writer.Append(context.Background(), event); err != nil {
			t.Fatalf("append event for %s: %v", sessionID, err)
		}
	}
}

func ticketEvent(
	t *testing.T,
	sessionID string,
	eventType EventType,
	timestamp time.Time,
	payload TicketEventPayload,
) Event {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode ticket payload: %v", err)
	}
	return Event{
		Type:      eventType,
		Payload:   data,
		Timestamp: timestamp,
		SessionID: sessionID,
	}
}

package monitor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/event"
)

type stubObserver struct {
	observations map[string]Observation
}

func (s stubObserver) Observe(sessionID string) (Observation, error) {
	obs, ok := s.observations[sessionID]
	if !ok {
		return Observation{SessionID: sessionID}, nil
	}
	return obs, nil
}

type stubClaimsReader struct {
	claims map[string]SessionClaims
}

func (s stubClaimsReader) ReadSession(sessionID string) (SessionClaims, error) {
	claim, ok := s.claims[sessionID]
	if !ok {
		return SessionClaims{SessionID: sessionID}, nil
	}
	return claim, nil
}

type stubTicketMaterializer struct {
	calledWith []string
}

func (s *stubTicketMaterializer) Materialize(_ context.Context, sessionIDs []string) error {
	s.calledWith = append([]string(nil), sessionIDs...)
	return nil
}

func TestMonitorRunOnceWritesSessionMetaAndTickets(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionA := "cook-a"
	sessionB := "cook-b"
	for _, sessionID := range []string{sessionA, sessionB} {
		if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions", sessionID), 0o755); err != nil {
			t.Fatalf("mkdir session directory: %v", err)
		}
	}

	now := time.Date(2026, 2, 22, 16, 0, 0, 0, time.UTC)
	monitor := NewMonitor(runtimeDir)
	monitor.now = func() time.Time { return now }
	monitor.observer = stubObserver{observations: map[string]Observation{
		sessionA: {
			SessionID: sessionA,
			Alive:     true,
			LogMTime:  now.Add(-30 * time.Second),
			LogSize:   100,
		},
		sessionB: {
			SessionID: sessionB,
			Alive:     false,
			LogMTime:  now.Add(-4 * time.Minute),
			LogSize:   80,
		},
	}}
	monitor.claims = stubClaimsReader{claims: map[string]SessionClaims{
		sessionA: {
			SessionID:    sessionA,
			HasEvents:    true,
			Provider:     "claude",
			FirstEventAt: now.Add(-3 * time.Minute),
			LastEventAt:  now.Add(-30 * time.Second),
			LastAction:   "apply patch",
			TotalCostUSD: 0.2,
		},
		sessionB: {
			SessionID:    sessionB,
			HasEvents:    true,
			Provider:     "codex",
			FirstEventAt: now.Add(-10 * time.Minute),
			LastEventAt:  now.Add(-4 * time.Minute),
			LastAction:   "run tests",
			TotalCostUSD: 0.05,
		},
	}}
	monitor.tickets = NewEventTicketMaterializer(runtimeDir)

	writeTicketEvent(t, runtimeDir, sessionA, event.EventTicketClaim, event.TicketEventPayload{
		Target:     "task-1",
		TargetType: event.TicketTargetBacklogItem,
	})
	writeTicketEvent(t, runtimeDir, sessionB, event.EventTicketClaim, event.TicketEventPayload{
		Target:     "task-2",
		TargetType: event.TicketTargetBacklogItem,
	})

	metas, err := monitor.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if len(metas) != 2 {
		t.Fatalf("meta count = %d", len(metas))
	}

	metaA := mustReadMeta(t, filepath.Join(runtimeDir, "sessions", sessionA, "meta.json"))
	if metaA.Status != SessionStatusRunning {
		t.Fatalf("session A status = %q", metaA.Status)
	}
	if metaA.CurrentAction != "apply patch" {
		t.Fatalf("session A action = %q", metaA.CurrentAction)
	}
	if metaA.TotalCostUSD != 0.2 {
		t.Fatalf("session A cost = %f", metaA.TotalCostUSD)
	}

	metaB := mustReadMeta(t, filepath.Join(runtimeDir, "sessions", sessionB, "meta.json"))
	if metaB.Status != SessionStatusFailed {
		t.Fatalf("session B status = %q", metaB.Status)
	}

	data, err := os.ReadFile(filepath.Join(runtimeDir, "tickets.json"))
	if err != nil {
		t.Fatalf("read tickets.json: %v", err)
	}
	var tickets []event.Ticket
	if err := json.Unmarshal(data, &tickets); err != nil {
		t.Fatalf("parse tickets.json: %v", err)
	}
	if len(tickets) != 2 {
		t.Fatalf("ticket count = %d", len(tickets))
	}
}

func writeTicketEvent(
	t *testing.T,
	runtimeDir string,
	sessionID string,
	eventType event.EventType,
	payload event.TicketEventPayload,
) {
	t.Helper()

	writer, err := event.NewEventWriter(runtimeDir, sessionID)
	if err != nil {
		t.Fatalf("new event writer: %v", err)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal ticket payload: %v", err)
	}
	if err := writer.Append(context.Background(), event.Event{
		Type:      eventType,
		Payload:   body,
		Timestamp: time.Date(2026, 2, 22, 15, 59, 0, 0, time.UTC),
		SessionID: sessionID,
	}); err != nil {
		t.Fatalf("append ticket event: %v", err)
	}
}

func mustReadMeta(t *testing.T, path string) SessionMeta {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read meta: %v", err)
	}
	var meta SessionMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("parse meta: %v", err)
	}
	return meta
}

func TestMonitorRunOncePassesSessionIDsToTickets(t *testing.T) {
	runtimeDir := t.TempDir()
	for _, sessionID := range []string{"cook-a", "cook-b"} {
		if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions", sessionID), 0o755); err != nil {
			t.Fatalf("mkdir session directory: %v", err)
		}
	}

	monitor := NewMonitor(runtimeDir)
	monitor.observer = stubObserver{}
	monitor.claims = stubClaimsReader{}
	stubTickets := &stubTicketMaterializer{}
	monitor.tickets = stubTickets

	if _, err := monitor.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if len(stubTickets.calledWith) != 2 {
		t.Fatalf("ticket materializer called with %d sessions", len(stubTickets.calledWith))
	}
}

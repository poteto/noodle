package monitor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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

package monitor

import (
	"testing"
	"time"
)

func TestDeriveSessionMetaRunningAndHealth(t *testing.T) {
	now := time.Date(2026, 2, 22, 15, 10, 0, 0, time.UTC)

	meta := DeriveSessionMeta(
		"cook-a",
		Observation{
			SessionID: "cook-a",
			Alive:     true,
			LogMTime:  now.Add(-10 * time.Minute),
			LogSize:   42,
		},
		SessionClaims{
			SessionID:    "cook-a",
			HasEvents:    true,
			Provider:     "claude",
			FirstEventAt: now.Add(-20 * time.Minute),
			LastEventAt:  now.Add(-10 * time.Minute),
			TokensIn:     500,
			TokensOut:    250,
		},
		SessionMeta{},
		now,
	)

	if meta.Status != SessionStatusRunning {
		t.Fatalf("status = %q", meta.Status)
	}
	if meta.Health != HealthGreen {
		t.Fatalf("health = %q", meta.Health)
	}
	if meta.Stuck {
		t.Fatal("expected stuck=false")
	}
	if meta.DurationSeconds <= 0 {
		t.Fatalf("duration seconds = %d", meta.DurationSeconds)
	}
	if meta.LastActivity.IsZero() {
		t.Fatal("last activity should be set")
	}
}

func TestDeriveSessionMetaExited(t *testing.T) {
	now := time.Date(2026, 2, 22, 15, 10, 0, 0, time.UTC)

	meta := DeriveSessionMeta(
		"cook-b",
		Observation{SessionID: "cook-b", Alive: false},
		SessionClaims{SessionID: "cook-b", HasEvents: true, Completed: true},
		SessionMeta{},
		now,
	)
	if meta.Status != SessionStatusExited {
		t.Fatalf("status = %q", meta.Status)
	}
	if meta.Health != HealthYellow {
		t.Fatalf("health = %q", meta.Health)
	}
}

func TestDeriveSessionMetaNoCompletionIsFailed(t *testing.T) {
	now := time.Date(2026, 2, 22, 15, 10, 0, 0, time.UTC)

	meta := DeriveSessionMeta(
		"cook-d",
		Observation{SessionID: "cook-d", Alive: false},
		SessionClaims{SessionID: "cook-d", HasEvents: false},
		SessionMeta{},
		now,
	)
	if meta.Status != SessionStatusFailed {
		t.Fatalf("status = %q", meta.Status)
	}
	if meta.Health != HealthRed {
		t.Fatalf("health = %q", meta.Health)
	}
}

func TestDeriveSessionMetaNoCompletionWithEventsIsExited(t *testing.T) {
	now := time.Date(2026, 2, 22, 15, 10, 0, 0, time.UTC)

	meta := DeriveSessionMeta(
		"cook-e",
		Observation{SessionID: "cook-e", Alive: false},
		SessionClaims{SessionID: "cook-e", HasEvents: true},
		SessionMeta{},
		now,
	)
	if meta.Status != SessionStatusExited {
		t.Fatalf("status = %q", meta.Status)
	}
	if meta.Health != HealthYellow {
		t.Fatalf("health = %q", meta.Health)
	}
}

func TestDeriveSessionMetaFailed(t *testing.T) {
	now := time.Date(2026, 2, 22, 15, 10, 0, 0, time.UTC)

	meta := DeriveSessionMeta(
		"cook-c",
		Observation{SessionID: "cook-c", Alive: false},
		SessionClaims{SessionID: "cook-c", HasEvents: true, Failed: true},
		SessionMeta{},
		now,
	)
	if meta.Status != SessionStatusFailed {
		t.Fatalf("status = %q", meta.Status)
	}
	if meta.Health != HealthRed {
		t.Fatalf("health = %q", meta.Health)
	}
}

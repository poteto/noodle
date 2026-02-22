package event

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/poteto/noodle/internal/testutil/fixturedir"
)

func TestTicketMaterializerDirectoryFixtures(t *testing.T) {
	fixturedir.AssertValidFixtureRoot(t, "testdata")
	inventory := fixturedir.LoadInventory(t, "testdata")
	for _, fixtureCase := range inventory.Cases {
		fixtureCase := fixtureCase
		t.Run(fixtureCase.Name, func(t *testing.T) {
			runTicketFixture(t, fixtureCase)
		})
	}
}

func TestTicketFixtureInventoryParity(t *testing.T) {
	expected := []string{"tickets"}
	inventory := fixturedir.LoadInventory(t, "testdata")
	if !reflect.DeepEqual(inventory.Names(), expected) {
		t.Fatalf("fixture inventory mismatch\\nactual:   %v\\nexpected: %v", inventory.Names(), expected)
	}
}

func runTicketFixture(t *testing.T, fixtureCase fixturedir.FixtureCase) {
	t.Helper()
	state := fixtureCase.States[0]

	var sessions map[string][]Event
	if err := json.Unmarshal(state.MustReadFile(t, "sessions.json"), &sessions); err != nil {
		t.Fatalf("decode sessions fixture: %v", err)
	}

	var options struct {
		Now            time.Time `json:"now"`
		Timeout        string    `json:"timeout"`
		ActiveSessions []string  `json:"active_sessions"`
	}
	if err := json.Unmarshal(state.MustReadFile(t, "options.json"), &options); err != nil {
		t.Fatalf("decode options fixture: %v", err)
	}
	timeout, err := time.ParseDuration(options.Timeout)
	if err != nil {
		t.Fatalf("parse timeout %q: %v", options.Timeout, err)
	}

	expectedRaw := fixturedir.MustSection(t, fixtureCase, "Expected Tickets")
	var expected []Ticket
	if err := json.Unmarshal([]byte(expectedRaw), &expected); err != nil {
		t.Fatalf("decode expected tickets: %v", err)
	}

	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	for sessionID, events := range sessions {
		writer, err := NewEventWriter(runtimeDir, sessionID)
		if err != nil {
			t.Fatalf("new writer %s: %v", sessionID, err)
		}
		for _, event := range events {
			if event.SessionID == "" {
				event.SessionID = sessionID
			}
			if err := writer.Append(context.Background(), event); err != nil {
				t.Fatalf("append fixture event for %s: %v", sessionID, err)
			}
		}
	}

	materializer := NewTicketMaterializer(runtimeDir)
	materializer.now = func() time.Time { return options.Now }
	materializer.staleTimeout = timeout

	actual, err := materializer.Materialize(context.Background(), options.ActiveSessions)
	if err != nil {
		t.Fatalf("materialize tickets: %v", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("fixture mismatch\nactual:   %#v\nexpected: %#v", actual, expected)
	}
}

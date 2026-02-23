package event

import (
	"context"
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

func runTicketFixture(t *testing.T, fixtureCase fixturedir.FixtureCase) {
	t.Helper()
	state := fixturedir.RequireSingleState(t, fixtureCase)

	sessions := fixturedir.ParseStateJSON[map[string][]Event](t, state, "sessions.json")
	options := fixturedir.ParseStateJSON[struct {
		Now            time.Time `json:"now"`
		Timeout        string    `json:"timeout"`
		ActiveSessions []string  `json:"active_sessions"`
	}](t, state, "options.json")
	timeout, err := time.ParseDuration(options.Timeout)
	if err != nil {
		t.Fatalf("parse timeout %q: %v", options.Timeout, err)
	}

	expected := fixturedir.ParseSectionJSON[[]Ticket](t, fixtureCase, "Expected Tickets")

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

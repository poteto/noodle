package monitor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/internal/testutil/fixturedir"
)

type monitorRunOnceFixtureInput struct {
	Now          time.Time                    `json:"now"`
	Observations map[string]Observation       `json:"observations"`
	Claims       map[string]SessionClaims     `json:"claims"`
	TicketEvents map[string][]event.Event     `json:"ticket_events"`
}

type monitorRunOnceFixtureDump struct {
	States map[string]monitorRunOnceStateDump `json:"states"`
}

type monitorRunOnceStateDump struct {
	ReturnedMetaCount   int                               `json:"returned_meta_count"`
	TicketCount         int                               `json:"ticket_count"`
	StatusBySession     map[string]SessionStatus          `json:"status_by_session,omitempty"`
	ActionBySession     map[string]string                 `json:"action_by_session,omitempty"`
	CostBySession       map[string]float64                `json:"cost_by_session,omitempty"`
	TicketStatusByTarget map[string]event.TicketStatus    `json:"ticket_status_by_target,omitempty"`
}

func TestRunOnceDirectoryFixtures(t *testing.T) {
	fixturedir.AssertValidFixtureRoot(t, "testdata")
	inventory := fixturedir.LoadInventory(t, "testdata")

	for _, fixtureCase := range inventory.Cases {
		if !strings.HasPrefix(fixtureCase.Name, "run-once-") {
			continue
		}
		fixtureCase := fixtureCase
		t.Run(fixtureCase.Name, func(t *testing.T) {
			expected := fixturedir.ParseSectionJSON[monitorRunOnceFixtureDump](t, fixtureCase, "Run Once Dump")
			observed := monitorRunOnceFixtureDump{
				States: make(map[string]monitorRunOnceStateDump, len(fixtureCase.States)),
			}

			runtimeDir := t.TempDir()
			monitor := NewMonitor(runtimeDir)

			for _, state := range fixtureCase.States {
				input := fixturedir.ParseStateJSON[monitorRunOnceFixtureInput](t, state, "input.json")
				ensureFixtureSessionDirs(t, runtimeDir, input)
				appendFixtureTicketEvents(t, runtimeDir, input.TicketEvents)

				monitor.now = func() time.Time { return input.Now }
				monitor.observer = stubObserver{observations: input.Observations}
				monitor.claims = stubClaimsReader{claims: input.Claims}
				monitor.tickets = NewEventTicketMaterializer(runtimeDir)

				metas, err := monitor.RunOnce(context.Background())
				if err != nil {
					t.Fatalf("run once state %s: %v", state.ID, err)
				}
				tickets := readFixtureTickets(t, runtimeDir)

				statusBySession := map[string]SessionStatus{}
				actionBySession := map[string]string{}
				costBySession := map[string]float64{}
				for _, meta := range metas {
					statusBySession[meta.SessionID] = meta.Status
					costBySession[meta.SessionID] = meta.TotalCostUSD
					if action := strings.TrimSpace(meta.CurrentAction); action != "" {
						actionBySession[meta.SessionID] = action
					}
				}

				ticketStatusByTarget := map[string]event.TicketStatus{}
				for _, ticket := range tickets {
					ticketStatusByTarget[ticket.Target] = ticket.Status
				}

				dump := monitorRunOnceStateDump{
					ReturnedMetaCount:    len(metas),
					TicketCount:          len(tickets),
					StatusBySession:      statusBySession,
					ActionBySession:      nilIfEmptyStringMap(actionBySession),
					CostBySession:        nilIfEmptyFloatMap(costBySession),
					TicketStatusByTarget: nilIfEmptyTicketStatusMap(ticketStatusByTarget),
				}
				if len(dump.StatusBySession) == 0 {
					dump.StatusBySession = nil
				}

				observed.States[state.ID] = dump
			}

			if !reflect.DeepEqual(observed, expected) {
				t.Fatalf("run once dump mismatch\nactual:\n%s\nexpected:\n%s", mustFixtureJSON(observed), mustFixtureJSON(expected))
			}
		})
	}
}

func ensureFixtureSessionDirs(t *testing.T, runtimeDir string, input monitorRunOnceFixtureInput) {
	t.Helper()
	sessionIDs := map[string]struct{}{}
	for sessionID := range input.Observations {
		sessionID = strings.TrimSpace(sessionID)
		if sessionID != "" {
			sessionIDs[sessionID] = struct{}{}
		}
	}
	for sessionID := range input.Claims {
		sessionID = strings.TrimSpace(sessionID)
		if sessionID != "" {
			sessionIDs[sessionID] = struct{}{}
		}
	}
	for sessionID := range input.TicketEvents {
		sessionID = strings.TrimSpace(sessionID)
		if sessionID != "" {
			sessionIDs[sessionID] = struct{}{}
		}
	}
	for sessionID := range sessionIDs {
		path := filepath.Join(runtimeDir, "sessions", sessionID)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir session dir %s: %v", sessionID, err)
		}
	}
}

func appendFixtureTicketEvents(t *testing.T, runtimeDir string, eventsBySession map[string][]event.Event) {
	t.Helper()
	for sessionID, events := range eventsBySession {
		sessionID = strings.TrimSpace(sessionID)
		if sessionID == "" {
			continue
		}
		writer, err := event.NewEventWriter(runtimeDir, sessionID)
		if err != nil {
			t.Fatalf("new event writer %s: %v", sessionID, err)
		}
		for _, ticketEvent := range events {
			if strings.TrimSpace(ticketEvent.SessionID) == "" {
				ticketEvent.SessionID = sessionID
			}
			if err := writer.Append(context.Background(), ticketEvent); err != nil {
				t.Fatalf("append ticket event for %s: %v", sessionID, err)
			}
		}
	}
}

func readFixtureTickets(t *testing.T, runtimeDir string) []event.Ticket {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(runtimeDir, "tickets.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read tickets.json: %v", err)
	}
	var tickets []event.Ticket
	if err := json.Unmarshal(data, &tickets); err != nil {
		t.Fatalf("parse tickets.json: %v", err)
	}
	return tickets
}

func nilIfEmptyStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	return input
}

func nilIfEmptyFloatMap(input map[string]float64) map[string]float64 {
	if len(input) == 0 {
		return nil
	}
	return input
}

func nilIfEmptyTicketStatusMap(input map[string]event.TicketStatus) map[string]event.TicketStatus {
	if len(input) == 0 {
		return nil
	}
	return input
}

func mustFixtureJSON(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

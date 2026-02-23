package event

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/filex"
)

const defaultTicketStaleTimeout = 30 * time.Minute

type ticketState struct {
	Ticket
}

// TicketMaterializer derives active tickets from session event logs.
type TicketMaterializer struct {
	runtimeDir   string
	staleTimeout time.Duration
	now          func() time.Time
	reader       *EventReader
}

// NewTicketMaterializer constructs a ticket materializer.
func NewTicketMaterializer(runtimeDir string) *TicketMaterializer {
	reader := NewEventReader(runtimeDir)
	return &TicketMaterializer{
		runtimeDir:   strings.TrimSpace(runtimeDir),
		staleTimeout: defaultTicketStaleTimeout,
		now:          time.Now,
		reader:       reader,
	}
}

// Materialize reads active session logs and writes tickets.json atomically.
//
// If activeSessionIDs is empty, all discovered session directories are used.
func (m *TicketMaterializer) Materialize(ctx context.Context, activeSessionIDs []string) ([]Ticket, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if strings.TrimSpace(m.runtimeDir) == "" {
		return nil, fmt.Errorf("runtime directory is required")
	}

	sessionIDs := uniqueTrimmed(activeSessionIDs)
	if len(sessionIDs) == 0 {
		discovered, err := discoverSessionIDs(m.runtimeDir)
		if err != nil {
			return nil, err
		}
		sessionIDs = discovered
	}

	allEvents := make([]Event, 0, 128)
	ticketFilter := EventFilter{Types: ticketEventTypes()}
	for _, sessionID := range sessionIDs {
		events, err := m.reader.ReadSession(sessionID, ticketFilter)
		if err != nil {
			return nil, fmt.Errorf("read ticket events for %s: %w", sessionID, err)
		}
		allEvents = append(allEvents, events...)
	}

	sort.SliceStable(allEvents, func(i, j int) bool {
		return allEvents[i].Timestamp.Before(allEvents[j].Timestamp)
	})

	tickets := materializeTickets(allEvents, m.now().UTC(), m.staleTimeout)
	if err := writeTicketsAtomic(ticketsPath(m.runtimeDir), tickets); err != nil {
		return nil, err
	}
	return tickets, nil
}

// ActiveTickets returns only non-stale tickets.
func ActiveTickets(tickets []Ticket) []Ticket {
	active := make([]Ticket, 0, len(tickets))
	for _, ticket := range tickets {
		if ticket.Status == TicketStatusStale {
			continue
		}
		active = append(active, ticket)
	}
	return active
}

func materializeTickets(events []Event, now time.Time, staleTimeout time.Duration) []Ticket {
	states := make(map[string]ticketState)

	for _, event := range events {
		payload, ok := parseTicketPayload(event)
		if !ok {
			continue
		}
		key, ok := resolveTicketKey(states, event, payload)
		if !ok {
			continue
		}

		switch event.Type {
		case EventTicketClaim:
			states[key] = ticketState{
				Ticket: Ticket{
					Target:       payload.Target,
					TargetType:   payload.TargetType,
					CookID:       event.SessionID,
					Files:        append([]string(nil), payload.Files...),
					ClaimedAt:    event.Timestamp,
					LastProgress: event.Timestamp,
					Status:       TicketStatusActive,
				},
			}
		case EventTicketProgress:
			state, exists := states[key]
			if !exists || state.CookID != event.SessionID {
				continue
			}
			state.LastProgress = event.Timestamp
			state.Status = TicketStatusActive
			state.BlockedBy = ""
			state.Reason = ""
			states[key] = state
		case EventTicketBlocked:
			state, exists := states[key]
			if !exists || state.CookID != event.SessionID {
				continue
			}
			state.Status = TicketStatusBlocked
			state.BlockedBy = payload.BlockedBy
			state.Reason = payload.Reason
			states[key] = state
		case EventTicketDone, EventTicketRelease:
			state, exists := states[key]
			if !exists || state.CookID != event.SessionID {
				continue
			}
			delete(states, key)
		}
	}

	tickets := make([]Ticket, 0, len(states))
	for _, state := range states {
		ticket := state.Ticket
		last := ticket.LastProgress
		if last.IsZero() {
			last = ticket.ClaimedAt
		}
		if staleTimeout > 0 &&
			!last.IsZero() &&
			now.Sub(last) > staleTimeout {
			ticket.Status = TicketStatusStale
		}
		tickets = append(tickets, ticket)
	}

	sort.SliceStable(tickets, func(i, j int) bool {
		if tickets[i].ClaimedAt.Equal(tickets[j].ClaimedAt) {
			if tickets[i].Target == tickets[j].Target {
				return tickets[i].CookID < tickets[j].CookID
			}
			return tickets[i].Target < tickets[j].Target
		}
		return tickets[i].ClaimedAt.Before(tickets[j].ClaimedAt)
	})
	return tickets
}

func parseTicketPayload(event Event) (TicketEventPayload, bool) {
	if !isTicketEventType(event.Type) || len(event.Payload) == 0 {
		return TicketEventPayload{}, false
	}
	var payload TicketEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return TicketEventPayload{}, false
	}
	payload.Target = strings.TrimSpace(payload.Target)
	if payload.Target == "" {
		return TicketEventPayload{}, false
	}
	if payload.TargetType != "" && !isValidTargetType(payload.TargetType) {
		return TicketEventPayload{}, false
	}
	if event.Type == EventTicketClaim && !isValidTargetType(payload.TargetType) {
		return TicketEventPayload{}, false
	}
	return payload, true
}

func resolveTicketKey(states map[string]ticketState, event Event, payload TicketEventPayload) (string, bool) {
	// Claims define the target type explicitly.
	if event.Type == EventTicketClaim {
		return ticketKey(payload.TargetType, payload.Target), true
	}

	// Non-claim events may provide target_type; if so, honor it.
	if payload.TargetType != "" {
		return ticketKey(payload.TargetType, payload.Target), true
	}

	// Otherwise, infer from the cook's existing claim for the same target.
	for key, state := range states {
		if state.Target != payload.Target {
			continue
		}
		if state.CookID == event.SessionID {
			return key, true
		}
	}
	return "", false
}

func ticketEventTypes() map[EventType]struct{} {
	return map[EventType]struct{}{
		EventTicketClaim:    {},
		EventTicketProgress: {},
		EventTicketDone:     {},
		EventTicketBlocked:  {},
		EventTicketRelease:  {},
	}
}

func isTicketEventType(eventType EventType) bool {
	_, ok := ticketEventTypes()[eventType]
	return ok
}

func isValidTargetType(targetType TicketTargetType) bool {
	switch targetType {
	case TicketTargetBacklogItem, TicketTargetFile, TicketTargetPlanPhase:
		return true
	default:
		return false
	}
}

func ticketKey(targetType TicketTargetType, target string) string {
	return string(targetType) + "\x00" + target
}

func discoverSessionIDs(runtimeDir string) ([]string, error) {
	sessionsPath := filepath.Join(runtimeDir, "sessions")
	entries, err := os.ReadDir(sessionsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read sessions directory: %w", err)
	}

	sessionIDs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sessionIDs = append(sessionIDs, entry.Name())
	}
	sort.Strings(sessionIDs)
	return sessionIDs, nil
}

func uniqueTrimmed(values []string) []string {
	set := make(map[string]struct{}, len(values))
	uniq := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := set[value]; ok {
			continue
		}
		set[value] = struct{}{}
		uniq = append(uniq, value)
	}
	sort.Strings(uniq)
	return uniq
}

func writeTicketsAtomic(path string, tickets []Ticket) error {
	data, err := json.Marshal(tickets)
	if err != nil {
		return fmt.Errorf("encode tickets: %w", err)
	}
	if err := filex.WriteFileAtomic(path, data); err != nil {
		return fmt.Errorf("replace tickets file: %w", err)
	}
	return nil
}

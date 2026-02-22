package monitor

import (
	"context"
	"strings"

	"github.com/poteto/noodle/event"
)

type EventTicketMaterializer struct {
	inner *event.TicketMaterializer
}

func NewEventTicketMaterializer(runtimeDir string) *EventTicketMaterializer {
	return &EventTicketMaterializer{
		inner: event.NewTicketMaterializer(strings.TrimSpace(runtimeDir)),
	}
}

func (m *EventTicketMaterializer) Materialize(ctx context.Context, sessionIDs []string) error {
	_, err := m.inner.Materialize(ctx, sessionIDs)
	return err
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/cmdmeta"
	"github.com/poteto/noodle/event"
	"github.com/spf13/cobra"
)

func newEventCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "event",
		Short: cmdmeta.Short("event"),
	}
	cmd.AddCommand(
		newEventEmitCmd(app),
	)
	return cmd
}

func newEventEmitCmd(app *App) *cobra.Command {
	var payload string
	var sessionID string
	cmd := &cobra.Command{
		Use:   "emit <type>",
		Short: cmdmeta.Short("event", "emit"),
		Args:  exactTrimmedArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtimeDir, err := app.RuntimeDir()
			if err != nil {
				return err
			}
			sid := strings.TrimSpace(sessionID)
			if sid != "" {
				return runSessionEventEmit(runtimeDir, sid, strings.TrimSpace(args[0]), payload)
			}
			return runEventEmit(runtimeDir, strings.TrimSpace(args[0]), payload)
		},
	}
	cmd.Flags().StringVar(&payload, "payload", "", "Event payload as JSON")
	cmd.Flags().StringVar(&sessionID, "session", "", "Session ID (writes to session event log instead of loop event log)")
	return cmd
}

func runSessionEventEmit(runtimeDir, sessionID, eventType, payload string) error {
	w, err := event.NewEventWriter(runtimeDir, sessionID)
	if err != nil {
		return err
	}

	ev := event.Event{
		Type:      event.EventType(eventType),
		SessionID: sessionID,
	}
	if payload != "" {
		if !json.Valid([]byte(payload)) {
			return fmt.Errorf("payload is not valid JSON")
		}
		ev.Payload = json.RawMessage(payload)
	}
	return w.Append(context.Background(), ev)
}

func runEventEmit(runtimeDir, eventType, payload string) error {
	w := event.NewLoopEventWriter(filepath.Join(runtimeDir, "loop-events.ndjson"))

	if payload == "" {
		return w.Emit(event.LoopEventType(eventType), nil)
	}

	if !json.Valid([]byte(payload)) {
		return fmt.Errorf("payload is not valid JSON")
	}
	return w.Emit(event.LoopEventType(eventType), json.RawMessage(payload))
}

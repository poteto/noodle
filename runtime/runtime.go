package runtime

import (
	"context"

	"github.com/poteto/noodle/dispatcher"
)

// SessionHandle is the loop's view of a running session. Any dispatcher.Session
// satisfies this interface — the runtime layer does not add new methods yet.
// Phase 8 adds VerdictPath(); for now this matches the methods the loop uses.
type SessionHandle interface {
	ID() string
	Status() string
	Done() <-chan struct{}
	TotalCost() float64
	Kill() error
}

// RecoveredSession is a pre-existing session found by Runtime.Recover().
type RecoveredSession struct {
	OrderID     string
	Handle      SessionHandle
	RuntimeName string
}

// Runtime abstracts dispatch and session lifecycle for a single execution
// platform (tmux, sprites, cursor). The loop holds a map of runtimes keyed
// by name and routes dispatches based on the stage's Runtime field.
type Runtime interface {
	// Start launches background goroutines (observation, heartbeat, etc.).
	Start(ctx context.Context) error

	// Dispatch creates a new agent session.
	Dispatch(ctx context.Context, req dispatcher.DispatchRequest) (SessionHandle, error)

	// Kill cancels an active session.
	Kill(handle SessionHandle) error

	// Recover discovers pre-existing sessions from a previous loop run.
	Recover(ctx context.Context) ([]RecoveredSession, error)

	// Health returns a channel of health events from the runtime's
	// observation layer. The runtime pushes events when session health
	// state changes (stuck, dead, idle, healthy).
	Health() <-chan HealthEvent

	// Close stops background goroutines and cleans up.
	Close() error
}

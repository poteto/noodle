package runtime

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/poteto/noodle/dispatcher"
)

// DispatchRequest is shared with the existing dispatcher boundary.
type DispatchRequest = dispatcher.DispatchRequest

// AgentController aliases dispatcher.AgentController so loop/ doesn't import dispatcher.
type AgentController = dispatcher.AgentController

// NoopController returns a controller that rejects all steering attempts.
var NoopController = dispatcher.NoopController

// SyncResult aliases dispatcher.SyncResult so loop/ doesn't import dispatcher.
type SyncResult = dispatcher.SyncResult

// SessionOutcome aliases dispatcher.SessionOutcome so loop/ doesn't import dispatcher.
type SessionOutcome = dispatcher.SessionOutcome

// SessionStatus aliases dispatcher.SessionStatus so loop/ doesn't import dispatcher.
type SessionStatus = dispatcher.SessionStatus

// Sync result type constants.
const (
	SyncResultTypeNone   = dispatcher.SyncResultTypeNone
	SyncResultTypeBranch = dispatcher.SyncResultTypeBranch
)

// Session status constants re-exported for loop/.
const (
	StatusCompleted = dispatcher.StatusCompleted
	StatusFailed    = dispatcher.StatusFailed
	StatusCancelled = dispatcher.StatusCancelled
	StatusKilled    = dispatcher.StatusKilled
)

// SessionHandle is a runtime-agnostic session contract.
type SessionHandle interface {
	ID() string
	Outcome() SessionOutcome
	Done() <-chan struct{}
	TotalCost() float64
	Terminate() error
	ForceKill() error
	VerdictPath() string
	Controller() AgentController
}

// RecoveredSession is discovered from a runtime during startup recovery.
type RecoveredSession struct {
	OrderID       string
	SessionHandle SessionHandle
	RuntimeName   string
	Reason        string
}

// Runtime dispatches, observes, and recovers sessions for one platform.
type Runtime interface {
	Dispatch(ctx context.Context, req DispatchRequest) (SessionHandle, error)
	Terminate(handle SessionHandle) error
	ForceKill(handle SessionHandle) error
	Recover(ctx context.Context) ([]RecoveredSession, error)
}

type dispatcherSessionHandle struct {
	session    dispatcher.Session
	runtimeDir string
}

func (s dispatcherSessionHandle) ID() string                  { return s.session.ID() }
func (s dispatcherSessionHandle) Outcome() SessionOutcome     { return s.session.Outcome() }
func (s dispatcherSessionHandle) Done() <-chan struct{}        { return s.session.Done() }
func (s dispatcherSessionHandle) TotalCost() float64    { return s.session.TotalCost() }
func (s dispatcherSessionHandle) Terminate() error      { return s.session.Terminate() }
func (s dispatcherSessionHandle) ForceKill() error      { return s.session.ForceKill() }
func (s dispatcherSessionHandle) VerdictPath() string {
	return filepath.Join(s.runtimeDir, "quality", s.session.ID()+".json")
}
func (s dispatcherSessionHandle) Controller() AgentController {
	return s.session.Controller()
}

// IsProcessStartFailure reports whether err is a typed process launch failure.
func IsProcessStartFailure(err error) bool {
	var startErr dispatcher.ProcessStartError
	return errors.As(err, &startErr)
}

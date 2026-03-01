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

// Sync result type constants.
const (
	SyncResultTypeNone   = dispatcher.SyncResultTypeNone
	SyncResultTypeBranch = dispatcher.SyncResultTypeBranch
)

// SessionHandle is a runtime-agnostic session contract.
type SessionHandle interface {
	ID() string
	Status() string
	Done() <-chan struct{}
	TotalCost() float64
	Kill() error
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
	Kill(handle SessionHandle) error
	Recover(ctx context.Context) ([]RecoveredSession, error)
}

type dispatcherSessionHandle struct {
	session    dispatcher.Session
	runtimeDir string
}

func (s dispatcherSessionHandle) ID() string            { return s.session.ID() }
func (s dispatcherSessionHandle) Status() string        { return s.session.Status() }
func (s dispatcherSessionHandle) Done() <-chan struct{} { return s.session.Done() }
func (s dispatcherSessionHandle) TotalCost() float64    { return s.session.TotalCost() }
func (s dispatcherSessionHandle) Kill() error           { return s.session.Kill() }
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

package runtime

import (
	"context"

	"github.com/poteto/noodle/dispatcher"
)

// TmuxRuntime wraps the existing TmuxDispatcher behind the Runtime interface.
type TmuxRuntime struct {
	dispatcher dispatcher.Dispatcher
	health     chan HealthEvent
}

func NewTmuxRuntime(d dispatcher.Dispatcher) *TmuxRuntime {
	return &TmuxRuntime{
		dispatcher: d,
		health:     make(chan HealthEvent, 256),
	}
}

func (r *TmuxRuntime) Start(_ context.Context) error { return nil }

func (r *TmuxRuntime) Dispatch(ctx context.Context, req dispatcher.DispatchRequest) (SessionHandle, error) {
	return r.dispatcher.Dispatch(ctx, req)
}

func (r *TmuxRuntime) Kill(handle SessionHandle) error {
	return handle.Kill()
}

func (r *TmuxRuntime) Recover(_ context.Context) ([]RecoveredSession, error) {
	return nil, nil
}

func (r *TmuxRuntime) Health() <-chan HealthEvent { return r.health }

func (r *TmuxRuntime) Close() error { return nil }

var _ Runtime = (*TmuxRuntime)(nil)

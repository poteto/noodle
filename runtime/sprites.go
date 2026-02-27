package runtime

import (
	"context"

	"github.com/poteto/noodle/dispatcher"
)

// SpritesRuntime wraps the existing SpritesDispatcher behind the Runtime interface.
type SpritesRuntime struct {
	dispatcher dispatcher.Dispatcher
	health     chan HealthEvent
}

func NewSpritesRuntime(d dispatcher.Dispatcher) *SpritesRuntime {
	return &SpritesRuntime{
		dispatcher: d,
		health:     make(chan HealthEvent, 256),
	}
}

func (r *SpritesRuntime) Start(_ context.Context) error { return nil }

func (r *SpritesRuntime) Dispatch(ctx context.Context, req dispatcher.DispatchRequest) (SessionHandle, error) {
	return r.dispatcher.Dispatch(ctx, req)
}

func (r *SpritesRuntime) Kill(handle SessionHandle) error {
	return handle.Kill()
}

func (r *SpritesRuntime) Recover(_ context.Context) ([]RecoveredSession, error) {
	return nil, nil
}

func (r *SpritesRuntime) Health() <-chan HealthEvent { return r.health }

func (r *SpritesRuntime) Close() error { return nil }

var _ Runtime = (*SpritesRuntime)(nil)

package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/poteto/noodle/dispatcher"
)

type DispatcherRuntime struct {
	name       string
	runtimeDir string
	dispatcher dispatcher.Dispatcher
	health     chan HealthEvent
}

func NewDispatcherRuntime(name string, d dispatcher.Dispatcher, runtimeDir string) *DispatcherRuntime {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		name = "tmux"
	}
	return &DispatcherRuntime{
		name:       name,
		runtimeDir: runtimeDir,
		dispatcher: d,
		health:     make(chan HealthEvent, 256),
	}
}

func (r *DispatcherRuntime) Start(context.Context) error { return nil }

func (r *DispatcherRuntime) Dispatch(ctx context.Context, req DispatchRequest) (SessionHandle, error) {
	if r == nil || r.dispatcher == nil {
		return nil, fmt.Errorf("runtime dispatcher not configured")
	}
	session, err := r.dispatcher.Dispatch(ctx, req)
	if err != nil {
		return nil, err
	}
	return dispatcherSessionHandle{session: session, runtimeDir: r.runtimeDir}, nil
}

func (r *DispatcherRuntime) Kill(handle SessionHandle) error {
	if handle == nil {
		return nil
	}
	return handle.Kill()
}

func (r *DispatcherRuntime) Recover(context.Context) ([]RecoveredSession, error) {
	return nil, nil
}

func (r *DispatcherRuntime) Health() <-chan HealthEvent {
	if r == nil {
		return nil
	}
	return r.health
}

func (r *DispatcherRuntime) Close() error { return nil }

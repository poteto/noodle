package dispatcher

import (
	"context"
	"fmt"
)

// DispatcherFactory routes dispatch requests to a runtime-specific dispatcher.
type DispatcherFactory struct {
	runtimes map[string]Dispatcher
}

func NewDispatcherFactory() *DispatcherFactory {
	return &DispatcherFactory{
		runtimes: map[string]Dispatcher{},
	}
}

func (f *DispatcherFactory) Register(runtime string, dispatcher Dispatcher) error {
	if f == nil {
		return fmt.Errorf("dispatcher factory not initialized")
	}
	if dispatcher == nil {
		return fmt.Errorf("dispatcher for runtime %q not configured", normalizeRuntime(runtime))
	}
	key := normalizeRuntime(runtime)
	f.runtimes[key] = dispatcher
	return nil
}

func (f *DispatcherFactory) Dispatch(ctx context.Context, req DispatchRequest) (Session, error) {
	if f == nil {
		return nil, fmt.Errorf("dispatcher factory not initialized")
	}
	runtime := normalizeRuntime(req.Runtime)
	dispatcher, ok := f.runtimes[runtime]
	if !ok {
		return nil, fmt.Errorf("runtime %q not configured", runtime)
	}
	req.Runtime = runtime
	return dispatcher.Dispatch(ctx, req)
}

var _ Dispatcher = (*DispatcherFactory)(nil)

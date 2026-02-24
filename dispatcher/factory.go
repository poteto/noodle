package dispatcher

import (
	"context"
	"fmt"
	"strings"
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

func (f *DispatcherFactory) Register(runtime string, dispatcher Dispatcher) {
	if f == nil || dispatcher == nil {
		return
	}
	key := normalizeFactoryRuntime(runtime)
	f.runtimes[key] = dispatcher
}

func (f *DispatcherFactory) Dispatch(ctx context.Context, req DispatchRequest) (Session, error) {
	if f == nil {
		return nil, fmt.Errorf("dispatcher factory not initialized")
	}
	runtime := normalizeFactoryRuntime(req.Runtime)
	dispatcher, ok := f.runtimes[runtime]
	if !ok {
		return nil, fmt.Errorf("runtime %q not configured", runtime)
	}
	req.Runtime = runtime
	return dispatcher.Dispatch(ctx, req)
}

func normalizeFactoryRuntime(runtime string) string {
	runtime = strings.ToLower(strings.TrimSpace(runtime))
	if runtime == "" {
		return "tmux"
	}
	return runtime
}

var _ Dispatcher = (*DispatcherFactory)(nil)

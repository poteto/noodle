package dispatcher

import (
	"context"
	"fmt"
	"os"
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
	session, err := dispatcher.Dispatch(ctx, req)
	if err != nil && runtime != "tmux" {
		fallback, hasFallback := f.runtimes["tmux"]
		if hasFallback {
			fmt.Fprintf(os.Stderr, "dispatch: %s runtime failed, falling back to tmux: %v\n", runtime, err)
			req.Runtime = "tmux"
			req.DispatchWarning = fmt.Sprintf("%s dispatch failed: %v", runtime, err)
			return fallback.Dispatch(ctx, req)
		}
	}
	return session, err
}

var _ Dispatcher = (*DispatcherFactory)(nil)

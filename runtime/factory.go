package runtime

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/poteto/noodle/dispatcher"
)

// RuntimeMap routes dispatch requests to the appropriate runtime based on
// the stage's Runtime field. Replaces the old DispatcherFactory.
type RuntimeMap struct {
	runtimes map[string]Runtime
}

func NewRuntimeMap() *RuntimeMap {
	return &RuntimeMap{runtimes: map[string]Runtime{}}
}

func (m *RuntimeMap) Register(name string, rt Runtime) {
	m.runtimes[strings.ToLower(strings.TrimSpace(name))] = rt
}

func (m *RuntimeMap) Get(name string) (Runtime, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		name = "tmux"
	}
	rt, ok := m.runtimes[name]
	return rt, ok
}

// Dispatch routes to the runtime specified in req.Runtime, falling back to
// tmux if the primary runtime fails (preserving existing factory semantics).
func (m *RuntimeMap) Dispatch(ctx context.Context, req dispatcher.DispatchRequest) (SessionHandle, error) {
	name := strings.ToLower(strings.TrimSpace(req.Runtime))
	if name == "" {
		name = "tmux"
	}
	rt, ok := m.runtimes[name]
	if !ok {
		return nil, fmt.Errorf("runtime %q not configured", name)
	}
	session, err := rt.Dispatch(ctx, req)
	if err != nil && name != "tmux" {
		if fallback, hasFallback := m.runtimes["tmux"]; hasFallback {
			fmt.Fprintf(os.Stderr, "dispatch: %s runtime failed, falling back to tmux: %v\n", name, err)
			req.Runtime = "tmux"
			req.DispatchWarning = fmt.Sprintf("%s dispatch failed: %v", name, err)
			return fallback.Dispatch(ctx, req)
		}
	}
	return session, err
}

// StartAll starts all registered runtimes.
func (m *RuntimeMap) StartAll(ctx context.Context) error {
	for name, rt := range m.runtimes {
		if err := rt.Start(ctx); err != nil {
			return fmt.Errorf("start runtime %s: %w", name, err)
		}
	}
	return nil
}

// CloseAll stops all registered runtimes.
func (m *RuntimeMap) CloseAll() {
	for _, rt := range m.runtimes {
		_ = rt.Close()
	}
}

// HealthChannels returns the health channel for each registered runtime.
func (m *RuntimeMap) HealthChannels() []<-chan HealthEvent {
	channels := make([]<-chan HealthEvent, 0, len(m.runtimes))
	for _, rt := range m.runtimes {
		channels = append(channels, rt.Health())
	}
	return channels
}

// RecoverAll calls Recover on each runtime and collects results.
func (m *RuntimeMap) RecoverAll(ctx context.Context) ([]RecoveredSession, error) {
	var all []RecoveredSession
	for name, rt := range m.runtimes {
		recovered, err := rt.Recover(ctx)
		if err != nil {
			return nil, fmt.Errorf("recover runtime %s: %w", name, err)
		}
		for i := range recovered {
			recovered[i].RuntimeName = name
		}
		all = append(all, recovered...)
	}
	return all, nil
}

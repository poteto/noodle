package runtime

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/poteto/noodle/dispatcher"
)

type DispatcherRuntime struct {
	name       string
	runtimeDir string
	dispatcher dispatcher.Dispatcher

	maxConcurrent int // 0 = unlimited
	mu            sync.Mutex
	active        int
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
	}
}

// SetMaxConcurrent sets the per-runtime concurrency cap. 0 means unlimited.
func (r *DispatcherRuntime) SetMaxConcurrent(n int) {
	r.mu.Lock()
	r.maxConcurrent = n
	r.mu.Unlock()
}

func (r *DispatcherRuntime) Dispatch(ctx context.Context, req DispatchRequest) (SessionHandle, error) {
	if r == nil || r.dispatcher == nil {
		return nil, fmt.Errorf("runtime dispatcher not configured")
	}

	r.mu.Lock()
	if r.maxConcurrent > 0 && r.active >= r.maxConcurrent {
		r.mu.Unlock()
		return nil, fmt.Errorf("runtime %s: concurrency limit reached (%d/%d)", r.name, r.active, r.maxConcurrent)
	}
	r.active++
	r.mu.Unlock()

	session, err := r.dispatcher.Dispatch(ctx, req)
	if err != nil {
		r.mu.Lock()
		r.active--
		r.mu.Unlock()
		return nil, err
	}

	handle := dispatcherSessionHandle{session: session, runtimeDir: r.runtimeDir}
	go r.watchDone(handle)
	return handle, nil
}

// watchDone decrements active count when the session completes.
func (r *DispatcherRuntime) watchDone(handle SessionHandle) {
	<-handle.Done()
	r.mu.Lock()
	r.active--
	r.mu.Unlock()
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

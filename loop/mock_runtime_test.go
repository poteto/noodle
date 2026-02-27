package loop

import (
	"context"
	"sync"

	loopruntime "github.com/poteto/noodle/runtime"
)

// mockRuntime implements runtime.Runtime for tests. It provides controllable
// dispatch behavior, configurable errors, and recovery support.
type mockRuntime struct {
	mu sync.Mutex

	calls       []loopruntime.DispatchRequest
	sessions    []*mockSession
	dispatchErr error
	recoverErr  error
	recovered   []loopruntime.RecoveredSession

	dispatchHook func(loopruntime.DispatchRequest) (loopruntime.SessionHandle, error)
}

func newMockRuntime() *mockRuntime {
	return &mockRuntime{}
}

func (m *mockRuntime) Dispatch(_ context.Context, req loopruntime.DispatchRequest) (loopruntime.SessionHandle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, req)

	if m.dispatchHook != nil {
		return m.dispatchHook(req)
	}
	if m.dispatchErr != nil {
		return nil, m.dispatchErr
	}

	s := &mockSession{
		id:     req.Name + "-id",
		status: "running",
		done:   make(chan struct{}),
	}
	m.sessions = append(m.sessions, s)
	return s, nil
}

func (m *mockRuntime) Kill(handle loopruntime.SessionHandle) error {
	if handle == nil {
		return nil
	}
	return handle.Kill()
}

func (m *mockRuntime) Recover(_ context.Context) ([]loopruntime.RecoveredSession, error) {
	if m.recoverErr != nil {
		return nil, m.recoverErr
	}
	return m.recovered, nil
}

// mockSession implements runtime.SessionHandle for tests.
type mockSession struct {
	id     string
	status string
	done   chan struct{}
	cost   float64
}

func (s *mockSession) ID() string            { return s.id }
func (s *mockSession) Status() string        { return s.status }
func (s *mockSession) Done() <-chan struct{} { return s.done }
func (s *mockSession) TotalCost() float64    { return s.cost }
func (s *mockSession) VerdictPath() string                     { return "" }
func (s *mockSession) Controller() loopruntime.AgentController { return loopruntime.NoopController() }

func (s *mockSession) Kill() error {
	s.status = "killed"
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	return nil
}

// complete transitions the session to the given status and closes Done().
func (s *mockSession) complete(status string) {
	s.status = status
	select {
	case <-s.done:
	default:
		close(s.done)
	}
}

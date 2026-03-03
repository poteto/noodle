package loop

import (
	"io"
	"log/slog"
	"sync/atomic"
	"testing"

	loopruntime "github.com/poteto/noodle/runtime"
)

type shutdownTestSession struct {
	id             string
	done           chan struct{}
	terminateCalls atomic.Int32
	forceKillCalls atomic.Int32
	onTerminate    func()
}

func newShutdownTestSession(id string) *shutdownTestSession {
	return &shutdownTestSession{
		id:   id,
		done: make(chan struct{}),
	}
}

func (s *shutdownTestSession) ID() string            { return s.id }
func (s *shutdownTestSession) Status() string        { return "running" }
func (s *shutdownTestSession) Done() <-chan struct{} { return s.done }
func (s *shutdownTestSession) TotalCost() float64    { return 0 }
func (s *shutdownTestSession) VerdictPath() string   { return "" }
func (s *shutdownTestSession) Controller() loopruntime.AgentController {
	return loopruntime.NoopController()
}

func (s *shutdownTestSession) Terminate() error {
	s.terminateCalls.Add(1)
	if s.onTerminate != nil {
		s.onTerminate()
	}
	return nil
}

func (s *shutdownTestSession) ForceKill() error {
	s.forceKillCalls.Add(1)
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	return nil
}

func TestShutdownEscalatesToForceKillWhenSessionStaysAlive(t *testing.T) {
	session := newShutdownTestSession("session-1")
	loop := &Loop{
		runtimeDir: t.TempDir(),
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		cooks: cookTracker{
			activeCooksByOrder: map[string]*cookHandle{
				"order-1": {session: session},
			},
		},
	}

	loop.Shutdown()

	if got := session.terminateCalls.Load(); got != 1 {
		t.Fatalf("terminate calls = %d, want 1", got)
	}
	if got := session.forceKillCalls.Load(); got != 1 {
		t.Fatalf("force kill calls = %d, want 1", got)
	}
	select {
	case <-session.done:
	default:
		t.Fatal("session should be marked done after force kill")
	}
}

func TestShutdownSkipsForceKillWhenTerminateCompletesSession(t *testing.T) {
	session := newShutdownTestSession("session-1")
	session.onTerminate = func() {
		select {
		case <-session.done:
		default:
			close(session.done)
		}
	}
	loop := &Loop{
		runtimeDir: t.TempDir(),
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		cooks: cookTracker{
			activeCooksByOrder: map[string]*cookHandle{
				"order-1": {session: session},
			},
		},
	}

	loop.Shutdown()

	if got := session.terminateCalls.Load(); got != 1 {
		t.Fatalf("terminate calls = %d, want 1", got)
	}
	if got := session.forceKillCalls.Load(); got != 0 {
		t.Fatalf("force kill calls = %d, want 0", got)
	}
}

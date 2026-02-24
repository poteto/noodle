package dispatcher

import (
	"context"
	"testing"
)

type factoryDispatcherStub struct {
	lastReq DispatchRequest
	session Session
}

func (s *factoryDispatcherStub) Dispatch(_ context.Context, req DispatchRequest) (Session, error) {
	s.lastReq = req
	return s.session, nil
}

type factorySessionStub struct {
	id     string
	status string
}

func (s *factorySessionStub) ID() string                  { return s.id }
func (s *factorySessionStub) Status() string              { return s.status }
func (s *factorySessionStub) Events() <-chan SessionEvent { return nil }
func (s *factorySessionStub) Done() <-chan struct{}       { return nil }
func (s *factorySessionStub) TotalCost() float64          { return 0 }
func (s *factorySessionStub) Kill() error                 { return nil }

var _ Session = (*factorySessionStub)(nil)

func TestDispatcherFactoryRoutesDefaultToTmux(t *testing.T) {
	factory := NewDispatcherFactory()
	stub := &factoryDispatcherStub{session: &factorySessionStub{id: "sess-1", status: "running"}}
	factory.Register("tmux", stub)

	session, err := factory.Dispatch(context.Background(), DispatchRequest{Name: "cook", Runtime: ""})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if session.ID() != "sess-1" {
		t.Fatalf("session ID = %q", session.ID())
	}
	if stub.lastReq.Runtime != "tmux" {
		t.Fatalf("runtime = %q, want tmux", stub.lastReq.Runtime)
	}
}

func TestDispatcherFactoryRoutesSpritesRuntime(t *testing.T) {
	factory := NewDispatcherFactory()
	stub := &factoryDispatcherStub{session: &factorySessionStub{id: "sess-2", status: "running"}}
	factory.Register("sprites", stub)

	_, err := factory.Dispatch(context.Background(), DispatchRequest{Name: "cook", Runtime: "sprites"})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if stub.lastReq.Runtime != "sprites" {
		t.Fatalf("runtime = %q, want sprites", stub.lastReq.Runtime)
	}
}

func TestDispatcherFactoryRejectsUnknownRuntime(t *testing.T) {
	factory := NewDispatcherFactory()
	factory.Register("tmux", &factoryDispatcherStub{
		session: &factorySessionStub{id: "sess-3", status: "running"},
	})

	if _, err := factory.Dispatch(context.Background(), DispatchRequest{Name: "cook", Runtime: "cursor"}); err == nil {
		t.Fatal("expected runtime configuration error")
	}
}

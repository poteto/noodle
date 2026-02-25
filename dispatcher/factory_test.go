package dispatcher

import (
	"context"
	"fmt"
	"strings"
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
	if err := factory.Register("tmux", stub); err != nil {
		t.Fatalf("register: %v", err)
	}

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
	if err := factory.Register("sprites", stub); err != nil {
		t.Fatalf("register: %v", err)
	}

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
	if err := factory.Register("tmux", &factoryDispatcherStub{
		session: &factorySessionStub{id: "sess-3", status: "running"},
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	if _, err := factory.Dispatch(context.Background(), DispatchRequest{Name: "cook", Runtime: "cursor"}); err == nil {
		t.Fatal("expected runtime configuration error")
	}
}

func TestDispatcherFactoryRegisterRejectsNilDispatcher(t *testing.T) {
	factory := NewDispatcherFactory()
	if err := factory.Register("tmux", nil); err == nil {
		t.Fatal("expected register error for nil dispatcher")
	}
}

type failingDispatcherStub struct {
	err error
}

func (s *failingDispatcherStub) Dispatch(_ context.Context, _ DispatchRequest) (Session, error) {
	return nil, s.err
}

func TestDispatcherFactoryFallsBackToTmuxOnRemoteFailure(t *testing.T) {
	factory := NewDispatcherFactory()
	tmuxStub := &factoryDispatcherStub{session: &factorySessionStub{id: "fallback-1", status: "running"}}
	spritesStub := &failingDispatcherStub{err: fmt.Errorf("clone on sprite: remote: Invalid username or token")}

	if err := factory.Register("tmux", tmuxStub); err != nil {
		t.Fatalf("register tmux: %v", err)
	}
	if err := factory.Register("sprites", spritesStub); err != nil {
		t.Fatalf("register sprites: %v", err)
	}

	session, err := factory.Dispatch(context.Background(), DispatchRequest{Name: "cook", Runtime: "sprites"})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if session.ID() != "fallback-1" {
		t.Fatalf("session ID = %q, want fallback-1", session.ID())
	}
	if tmuxStub.lastReq.Runtime != "tmux" {
		t.Fatalf("runtime = %q, want tmux", tmuxStub.lastReq.Runtime)
	}
	if !strings.Contains(tmuxStub.lastReq.DispatchWarning, "sprites dispatch failed") {
		t.Fatalf("dispatch warning = %q, want sprites dispatch failed", tmuxStub.lastReq.DispatchWarning)
	}
}

func TestDispatcherFactoryNoFallbackForTmuxFailure(t *testing.T) {
	factory := NewDispatcherFactory()
	tmuxStub := &failingDispatcherStub{err: fmt.Errorf("tmux not found")}

	if err := factory.Register("tmux", tmuxStub); err != nil {
		t.Fatalf("register: %v", err)
	}

	_, err := factory.Dispatch(context.Background(), DispatchRequest{Name: "cook", Runtime: "tmux"})
	if err == nil {
		t.Fatal("expected tmux dispatch error")
	}
}

func TestDispatcherFactoryNoFallbackWithoutTmux(t *testing.T) {
	factory := NewDispatcherFactory()
	spritesStub := &failingDispatcherStub{err: fmt.Errorf("clone failed")}

	if err := factory.Register("sprites", spritesStub); err != nil {
		t.Fatalf("register: %v", err)
	}

	_, err := factory.Dispatch(context.Background(), DispatchRequest{Name: "cook", Runtime: "sprites"})
	if err == nil {
		t.Fatal("expected sprites dispatch error with no tmux fallback")
	}
}

package server

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/poteto/noodle/event"
)

type mockSubscriber struct {
	mu       sync.Mutex
	messages [][]byte
	slowMode bool
	closed   bool
}

func (m *mockSubscriber) Send(msg []byte) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.slowMode {
		return false
	}
	m.messages = append(m.messages, msg)
	return true
}

func (m *mockSubscriber) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
}

func (m *mockSubscriber) messageCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

func (m *mockSubscriber) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func TestBrokerSubscribePublish(t *testing.T) {
	b := NewSessionEventBroker()
	sub1 := &mockSubscriber{}
	sub2 := &mockSubscriber{}
	b.Subscribe("session-1", sub1)
	b.Subscribe("session-1", sub2)

	b.PublishSessionEvent("session-1", event.Event{
		Type:      event.EventAction,
		Payload:   json.RawMessage(`{"tool":"read","message":"reading file"}`),
		Timestamp: time.Now(),
		SessionID: "session-1",
	})

	if sub1.messageCount() != 1 {
		t.Fatalf("sub1 got %d messages, want 1", sub1.messageCount())
	}
	if sub2.messageCount() != 1 {
		t.Fatalf("sub2 got %d messages, want 1", sub2.messageCount())
	}
}

func TestBrokerUnsubscribe(t *testing.T) {
	b := NewSessionEventBroker()
	sub := &mockSubscriber{}
	b.Subscribe("s1", sub)
	b.Unsubscribe("s1", sub)

	b.PublishSessionEvent("s1", event.Event{
		Type:      event.EventAction,
		Payload:   json.RawMessage(`{"tool":"read","message":"test"}`),
		Timestamp: time.Now(),
		SessionID: "s1",
	})

	if sub.messageCount() != 0 {
		t.Fatalf("got %d messages after unsubscribe, want 0", sub.messageCount())
	}
}

func TestBrokerUnsubscribeAll(t *testing.T) {
	b := NewSessionEventBroker()
	sub := &mockSubscriber{}
	b.Subscribe("s1", sub)
	b.Subscribe("s2", sub)
	b.UnsubscribeAll(sub)

	b.PublishSessionEvent("s1", event.Event{Type: event.EventAction, Payload: json.RawMessage(`{"tool":"read","message":"x"}`), Timestamp: time.Now(), SessionID: "s1"})
	b.PublishSessionEvent("s2", event.Event{Type: event.EventAction, Payload: json.RawMessage(`{"tool":"read","message":"x"}`), Timestamp: time.Now(), SessionID: "s2"})

	if sub.messageCount() != 0 {
		t.Fatalf("got %d messages, want 0", sub.messageCount())
	}
}

func TestBrokerSlowClientDisconnected(t *testing.T) {
	b := NewSessionEventBroker()
	slow := &mockSubscriber{slowMode: true}
	b.Subscribe("s1", slow)

	b.PublishSessionEvent("s1", event.Event{
		Type:      event.EventAction,
		Payload:   json.RawMessage(`{"tool":"bash","message":"running"}`),
		Timestamp: time.Now(),
		SessionID: "s1",
	})

	if !slow.isClosed() {
		t.Fatal("slow client should have been disconnected")
	}
	// Subsequent publishes should not reach the slow client.
	slow.slowMode = false
	b.PublishSessionEvent("s1", event.Event{
		Type:      event.EventAction,
		Payload:   json.RawMessage(`{"tool":"bash","message":"running"}`),
		Timestamp: time.Now(),
		SessionID: "s1",
	})
	if slow.messageCount() != 0 {
		t.Fatal("disconnected client should not receive messages")
	}
}

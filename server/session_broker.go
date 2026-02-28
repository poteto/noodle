package server

import (
	"encoding/json"
	"sync"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/internal/snapshot"
)

// Subscriber receives serialized messages. Decoupled from wsClient.
type Subscriber interface {
	Send(msg []byte) bool // false if client is slow/dead
	Close()               // disconnect the client
}

// SessionEventBroker implements dispatcher.SessionEventSink.
// Fans out session events to subscribed WebSocket clients.
type SessionEventBroker struct {
	mu          sync.RWMutex
	subscribers map[string]map[Subscriber]struct{}
}

func NewSessionEventBroker() *SessionEventBroker {
	return &SessionEventBroker{
		subscribers: make(map[string]map[Subscriber]struct{}),
	}
}

func (b *SessionEventBroker) Subscribe(sessionID string, sub Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs, ok := b.subscribers[sessionID]
	if !ok {
		subs = make(map[Subscriber]struct{})
		b.subscribers[sessionID] = subs
	}
	subs[sub] = struct{}{}
}

func (b *SessionEventBroker) Unsubscribe(sessionID string, sub Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if subs, ok := b.subscribers[sessionID]; ok {
		delete(subs, sub)
		if len(subs) == 0 {
			delete(b.subscribers, sessionID)
		}
	}
}

func (b *SessionEventBroker) UnsubscribeAll(sub Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for sessionID, subs := range b.subscribers {
		delete(subs, sub)
		if len(subs) == 0 {
			delete(b.subscribers, sessionID)
		}
	}
}

// PublishSessionEvent implements dispatcher.SessionEventSink.
// Converts event.Event -> EventLine, serializes, fans out.
func (b *SessionEventBroker) PublishSessionEvent(sessionID string, ev event.Event) {
	el, ok := snapshot.FormatSingleEvent(ev)
	if !ok {
		return
	}
	msg, err := json.Marshal(map[string]any{
		"type":       "session_event",
		"session_id": sessionID,
		"data":       el,
	})
	if err != nil {
		return
	}
	b.mu.RLock()
	subs := b.subscribers[sessionID]
	// Copy slice under read lock to avoid holding lock during Send.
	targets := make([]Subscriber, 0, len(subs))
	for sub := range subs {
		targets = append(targets, sub)
	}
	b.mu.RUnlock()

	for _, sub := range targets {
		if !sub.Send(msg) {
			// Slow client — disconnect, don't silently drop.
			b.Unsubscribe(sessionID, sub)
			sub.Close()
		}
	}
}

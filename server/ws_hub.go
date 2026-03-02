package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"github.com/poteto/noodle/internal/snapshot"
	"github.com/poteto/noodle/loop"
)

// wsClient is a connected WebSocket consumer. Implements Subscriber.
type wsClient struct {
	conn      *websocket.Conn
	send      chan []byte
	hub       *wsHub
	closeOnce sync.Once
}

// Send implements Subscriber. Non-blocking channel send; returns false if full
// or if the client has been closed.
func (c *wsClient) Send(msg []byte) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	select {
	case c.send <- msg:
		return true
	default:
		return false
	}
}

// Close implements Subscriber. Uses closeOnce to safely remove and close.
func (c *wsClient) Close() {
	c.closeOnce.Do(func() {
		c.hub.removeClient(c)
		close(c.send)
		c.conn.Close()
	})
}

// writePump drains the send channel to the WebSocket connection.
func (c *wsClient) writePump() {
	defer c.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

// readPump reads client messages and dispatches by type.
func (c *wsClient) readPump(s *Server) {
	defer c.Close()
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var msg struct {
			Type      string `json:"type"`
			SessionID string `json:"session_id"`
		}
		if json.Unmarshal(raw, &msg) != nil {
			continue
		}
		switch msg.Type {
		case "subscribe":
			if msg.SessionID == "" {
				continue
			}
			s.ws.broker.Subscribe(msg.SessionID, c)
			// Read backfill from disk.
			events, err := snapshot.ReadSessionEvents(s.runtimeDir, msg.SessionID)
			if err == nil && len(events) > 0 {
				backfill, _ := json.Marshal(map[string]any{
					"type":       "backfill",
					"session_id": msg.SessionID,
					"data":       events,
				})
				c.Send(backfill)
			}
			ack, _ := json.Marshal(map[string]any{
				"type":       "subscribed",
				"session_id": msg.SessionID,
			})
			c.Send(ack)

		case "unsubscribe":
			if msg.SessionID == "" {
				continue
			}
			s.ws.broker.Unsubscribe(msg.SessionID, c)
			ack, _ := json.Marshal(map[string]any{
				"type":       "unsubscribed",
				"session_id": msg.SessionID,
			})
			c.Send(ack)

		case "control":
			var ctrl struct {
				Type string         `json:"type"`
				Data controlRequest `json:"data"`
			}
			if json.Unmarshal(raw, &ctrl) != nil {
				continue
			}
			result, err := s.processControl(ctrl.Data)
			var resp map[string]any
			if err != nil {
				resp = map[string]any{"type": "control_error", "error": err.Error()}
			} else {
				resp = map[string]any{"type": "control_ack", "data": result}
			}
			ack, _ := json.Marshal(resp)
			c.Send(ack)
		}
	}
}

// wsHub manages WebSocket clients and broadcasts snapshot updates.
type wsHub struct {
	mu       sync.RWMutex
	clients  map[*wsClient]struct{}
	broker   *SessionEventBroker
	lastHash [sha256.Size]byte
	done     chan struct{}
}

func newWSHub(broker *SessionEventBroker) *wsHub {
	return &wsHub{
		clients: make(map[*wsClient]struct{}),
		broker:  broker,
		done:    make(chan struct{}),
	}
}

func (h *wsHub) addClient(c *wsClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
}

func (h *wsHub) removeClient(c *wsClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
}

// broadcastSnapshot sends data to all connected clients.
// Non-blocking send; drops messages for slow clients.
func (h *wsHub) broadcastSnapshot(data []byte) {
	h.mu.RLock()
	targets := make([]*wsClient, 0, len(h.clients))
	for c := range h.clients {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	for _, c := range targets {
		select {
		case c.send <- data:
		default:
			// Client too slow; drop message.
		}
	}
}

// watchAndBroadcast watches the runtime directory for changes and broadcasts
// snapshot updates to WebSocket clients. Debounces rapid filesystem events.
func (h *wsHub) watchAndBroadcast(ctx context.Context, runtimeDir string, now func() time.Time, provider LoopStateProvider, warnings []string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer watcher.Close()

	// Recursively watch the runtime directory and all subdirectories.
	filepath.WalkDir(runtimeDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		_ = watcher.Add(path)
		return nil
	})

	const debounce = 300 * time.Millisecond
	timer := time.NewTimer(debounce)
	if !timer.Stop() {
		<-timer.C
	}
	pending := false

	// Send initial snapshot.
	h.loadAndBroadcast(runtimeDir, now, provider, warnings)

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.done:
			return
		case ev, ok := <-watcher.Events:
			if !ok {
				return
			}
			// Watch newly created directories.
			if ev.Has(fsnotify.Create) {
				if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
					_ = watcher.Add(ev.Name)
				}
			}
			if !pending {
				timer.Reset(debounce)
				pending = true
			}
		case <-watcher.Errors:
			// Ignore watch errors.
		case <-timer.C:
			pending = false
			h.loadAndBroadcast(runtimeDir, now, provider, warnings)
		}
	}
}

// loadAndBroadcast loads a snapshot, diffs against the last hash, and
// broadcasts if changed.
func (h *wsHub) loadAndBroadcast(runtimeDir string, now func() time.Time, provider LoopStateProvider, warnings []string) {
	if provider == nil {
		return
	}
	snap, err := snapshot.LoadSnapshot(runtimeDir, now(), provider.State())
	if err != nil {
		return
	}
	snap.Warnings = warnings

	// Zero volatile field for diff-gating so UpdatedAt doesn't defeat the hash.
	hashSnap := snap
	hashSnap.UpdatedAt = time.Time{}
	hashData, err := json.Marshal(hashSnap)
	if err != nil {
		return
	}
	hash := sha256.Sum256(hashData)

	h.mu.Lock()
	same := hash == h.lastHash
	if !same {
		h.lastHash = hash
	}
	h.mu.Unlock()
	if same {
		return
	}

	// Broadcast the full snapshot as a JSON envelope.
	data, err := json.Marshal(snap)
	if err != nil {
		return
	}
	msg := fmt.Appendf(nil, `{"type":"snapshot","data":%s}`, data)
	h.broadcastSnapshot(msg)
}

// Close closes all client connections. Required because http.Server.Shutdown
// does not clean up hijacked WebSocket connections.
func (h *wsHub) Close() {
	select {
	case <-h.done:
	default:
		close(h.done)
	}

	h.mu.RLock()
	clients := make([]*wsClient, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		c.Close()
	}
}

// loadInitialSnapshot builds a snapshot for sending to a newly connected client.
func (h *wsHub) loadInitialSnapshot(runtimeDir string, now func() time.Time, provider LoopStateProvider, warnings []string) ([]byte, error) {
	if provider == nil {
		return nil, fmt.Errorf("no loop state provider")
	}
	snap, err := snapshot.LoadSnapshot(runtimeDir, now(), provider.State())
	if err != nil {
		return nil, err
	}
	snap.Warnings = warnings
	data, err := json.Marshal(snap)
	if err != nil {
		return nil, err
	}
	return fmt.Appendf(nil, `{"type":"snapshot","data":%s}`, data), nil
}

// upgrader allows localhost origins for dev.
var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		return isLocalhostOrigin(origin)
	},
}

func parseControlRequest(action string, req controlRequest) (loop.ControlCommand, error) {
	validActions := map[string]bool{
		"pause": true, "resume": true, "drain": true, "skip": true,
		"kill": true, "steer": true, "merge": true, "reject": true,
		"request-changes": true, "mode": true, "enqueue": true,
		"stop-all": true, "requeue": true, "edit-item": true,
		"reorder": true, "stop": true, "set-max-concurrency": true,
	}
	if !validActions[action] {
		return loop.ControlCommand{}, fmt.Errorf("unknown action: %s", action)
	}
	return loop.ControlCommand{
		Action:   action,
		OrderID:  strings.TrimSpace(req.OrderID),
		Name:     strings.TrimSpace(req.Name),
		Target:   strings.TrimSpace(req.Target),
		Prompt:   strings.TrimSpace(req.Prompt),
		Value:    strings.TrimSpace(req.Value),
		TaskKey:  strings.TrimSpace(req.TaskKey),
		Provider: strings.TrimSpace(req.Provider),
		Model:    strings.TrimSpace(req.Model),
		Skill:    strings.TrimSpace(req.Skill),
	}, nil
}

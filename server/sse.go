package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/poteto/noodle/internal/snapshot"
)

// sseClient is a connected SSE consumer.
type sseClient struct {
	ch     chan []byte
	closed chan struct{}
}

// sseHub manages SSE clients and broadcasts snapshot updates.
type sseHub struct {
	mu       sync.Mutex
	clients  map[*sseClient]struct{}
	lastHash [sha256.Size]byte
	done     chan struct{}
}

func newSSEHub() *sseHub {
	return &sseHub{
		clients: make(map[*sseClient]struct{}),
		done:    make(chan struct{}),
	}
}

func (h *sseHub) addClient(c *sseClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
}

func (h *sseHub) removeClient(c *sseClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
	close(c.closed)
}

func (h *sseHub) close() {
	select {
	case <-h.done:
	default:
		close(h.done)
	}
}

// broadcast sends data to all connected clients. Drops messages for slow clients.
func (h *sseHub) broadcast(data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		select {
		case c.ch <- data:
		default:
			// Client too slow; drop message.
		}
	}
}

// watchAndBroadcast watches the runtime directory for changes and broadcasts
// snapshot updates to SSE clients. Debounces rapid filesystem events.
func (h *sseHub) watchAndBroadcast(ctx context.Context, runtimeDir string, now func() time.Time) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer watcher.Close()

	// Watch the runtime directory for file changes.
	_ = watcher.Add(runtimeDir)

	// Also watch sessions/ subdirectory if it exists.
	sessionsDir := filepath.Join(runtimeDir, "sessions")
	_ = watcher.Add(sessionsDir)

	const debounce = 300 * time.Millisecond
	timer := time.NewTimer(debounce)
	if !timer.Stop() {
		<-timer.C
	}
	pending := false

	// Send initial snapshot.
	h.loadAndBroadcast(runtimeDir, now)

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.done:
			return
		case _, ok := <-watcher.Events:
			if !ok {
				return
			}
			if !pending {
				timer.Reset(debounce)
				pending = true
			}
		case <-watcher.Errors:
			// Ignore watch errors.
		case <-timer.C:
			pending = false
			h.loadAndBroadcast(runtimeDir, now)
		}
	}
}

// loadAndBroadcast loads a snapshot, diffs against the last hash, and
// broadcasts if changed.
func (h *sseHub) loadAndBroadcast(runtimeDir string, now func() time.Time) {
	snap, err := snapshot.LoadSnapshot(runtimeDir, now())
	if err != nil {
		return
	}
	data, err := json.Marshal(snap)
	if err != nil {
		return
	}
	hash := sha256.Sum256(data)
	h.mu.Lock()
	same := hash == h.lastHash
	if !same {
		h.lastHash = hash
	}
	h.mu.Unlock()
	if same {
		return
	}

	// Format as SSE event.
	msg := fmt.Appendf(nil, "event: snapshot\ndata: %s\n\n", data)
	h.broadcast(msg)
}

// handleSSE is the HTTP handler for GET /api/events.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	client := &sseClient{
		ch:     make(chan []byte, 16),
		closed: make(chan struct{}),
	}
	s.sse.addClient(client)
	defer s.sse.removeClient(client)

	// Send current snapshot immediately.
	snap, err := snapshot.LoadSnapshot(s.runtimeDir, s.now())
	if err == nil {
		data, err := json.Marshal(snap)
		if err == nil {
			fmt.Fprintf(w, "event: snapshot\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.sse.done:
			return
		case msg := <-client.ch:
			_, err := w.Write(msg)
			if err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

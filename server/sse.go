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
func (h *sseHub) watchAndBroadcast(ctx context.Context, runtimeDir string, now func() time.Time, provider LoopStateProvider) {
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
	h.loadAndBroadcast(runtimeDir, now, provider)

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
			h.loadAndBroadcast(runtimeDir, now, provider)
		}
	}
}

// loadAndBroadcast loads a snapshot, diffs against the last hash, and
// broadcasts if changed.
func (h *sseHub) loadAndBroadcast(runtimeDir string, now func() time.Time, provider LoopStateProvider) {
	if provider == nil {
		return
	}
	snap, err := snapshot.LoadSnapshot(runtimeDir, now(), provider.State())
	if err != nil {
		return
	}

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

	// Broadcast the full snapshot (with UpdatedAt).
	data, err := json.Marshal(snap)
	if err != nil {
		return
	}
	msg := fmt.Appendf(nil, "data: %s\n\n", data)
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
	snap, err := s.loadSnapshot()
	if err == nil {
		data, err := json.Marshal(snap)
		if err == nil {
			fmt.Fprintf(w, "data: %s\n\n", data)
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

package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/snapshot"
	"github.com/poteto/noodle/loop"
	"github.com/poteto/noodle/worktree"
)

//go:embed placeholder.html
var placeholderHTML []byte

// Options configures the HTTP server.
type Options struct {
	RuntimeDir string
	Addr       string // host:port, defaults to "127.0.0.1:0"
	Now        func() time.Time
	UI         fs.FS          // embedded SPA assets; nil = placeholder only
	Config     *config.Config // project config; nil = zero config
}

// Server serves the web UI API.
type Server struct {
	runtimeDir string
	now        func() time.Time
	httpServer *http.Server
	listener   net.Listener
	sse        *sseHub
	config     config.Config
	ready      chan struct{}
}

// New creates a Server but does not start it.
func New(opts Options) *Server {
	runtimeDir := strings.TrimSpace(opts.RuntimeDir)
	if runtimeDir == "" {
		runtimeDir = ".noodle"
	}
	addr := strings.TrimSpace(opts.Addr)
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}

	var cfg config.Config
	if opts.Config != nil {
		cfg = *opts.Config
	}

	s := &Server{
		runtimeDir: runtimeDir,
		now:        now,
		sse:        newSSEHub(),
		config:     cfg,
		ready:      make(chan struct{}),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/events", s.handleSSE)
	mux.HandleFunc("GET /api/snapshot", s.handleSnapshot)
	mux.HandleFunc("GET /api/sessions/{id}/events", s.handleSessionEvents)
	mux.HandleFunc("POST /api/control", s.handleControl)
	mux.HandleFunc("GET /api/config", s.handleConfig)
	mux.HandleFunc("GET /api/reviews/{id}/diff", s.handleReviewDiff)
	mux.Handle("GET /", uiOrPlaceholder(opts.UI))

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: corsMiddleware(mux),
	}
	return s
}

// FindPort tries to listen on startPort, incrementing up to 10 times if busy.
// Returns the addr string "127.0.0.1:PORT" that succeeded, or an error.
func FindPort(startPort int) (string, error) {
	const maxAttempts = 10
	for i := 0; i < maxAttempts; i++ {
		addr := fmt.Sprintf("127.0.0.1:%d", startPort+i)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			continue
		}
		ln.Close()
		return addr, nil
	}
	return "", fmt.Errorf("no available port in range %d-%d", startPort, startPort+maxAttempts-1)
}

// Start begins listening and serving. It starts the SSE file watcher in the
// background. Blocks until the server shuts down.
func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s.listener = ln
	close(s.ready)

	go s.sse.watchAndBroadcast(ctx, s.runtimeDir, s.now)

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.httpServer.Shutdown(shutdownCtx)
		s.sse.close()
		return nil
	case err := <-errCh:
		s.sse.close()
		return err
	}
}

// Addr returns the listener address. Only valid after Start begins.
func (s *Server) Addr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// WaitReady blocks until the server is listening.
func (s *Server) WaitReady() { <-s.ready }

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	snap, err := snapshot.LoadSnapshot(s.runtimeDir, s.now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func (s *Server) handleSessionEvents(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if strings.TrimSpace(sessionID) == "" {
		http.Error(w, "session ID required", http.StatusBadRequest)
		return
	}
	snap, err := snapshot.LoadSnapshot(s.runtimeDir, s.now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	events, ok := snap.EventsBySession[sessionID]
	if !ok {
		writeJSON(w, http.StatusOK, []snapshot.EventLine{})
		return
	}

	// Support ?after= for incremental fetches.
	if after := r.URL.Query().Get("after"); after != "" {
		ts, err := time.Parse(time.RFC3339Nano, after)
		if err == nil {
			filtered := make([]snapshot.EventLine, 0, len(events))
			for _, ev := range events {
				if ev.At.After(ts) {
					filtered = append(filtered, ev)
				}
			}
			events = filtered
		}
	}

	writeJSON(w, http.StatusOK, events)
}

// controlRequest is the JSON body for POST /api/control.
type controlRequest struct {
	Action   string `json:"action"`
	Item     string `json:"item,omitempty"`
	Name     string `json:"name,omitempty"`
	Target   string `json:"target,omitempty"`
	Prompt   string `json:"prompt,omitempty"`
	Value    string `json:"value,omitempty"`
	TaskKey  string `json:"task_key,omitempty"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	Skill    string `json:"skill,omitempty"`
}

func (s *Server) handleControl(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024) // 64KB limit

	var req controlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	action := strings.TrimSpace(req.Action)
	if action == "" {
		http.Error(w, "action required", http.StatusBadRequest)
		return
	}

	validActions := map[string]bool{
		"pause": true, "resume": true, "drain": true, "skip": true,
		"kill": true, "steer": true, "merge": true, "reject": true,
		"request-changes": true, "autonomy": true, "enqueue": true,
		"stop-all": true, "requeue": true, "edit-item": true,
		"reorder": true, "stop": true, "set-max-cooks": true,
	}
	if !validActions[action] {
		http.Error(w, "unknown action: "+action, http.StatusBadRequest)
		return
	}

	cmd := loop.ControlCommand{
		Action:   action,
		Item:     strings.TrimSpace(req.Item),
		Name:     strings.TrimSpace(req.Name),
		Target:   strings.TrimSpace(req.Target),
		Prompt:   strings.TrimSpace(req.Prompt),
		Value:    strings.TrimSpace(req.Value),
		TaskKey:  strings.TrimSpace(req.TaskKey),
		Provider: strings.TrimSpace(req.Provider),
		Model:    strings.TrimSpace(req.Model),
		Skill:    strings.TrimSpace(req.Skill),
		At:       s.now().UTC(),
	}
	cmd.ID = fmt.Sprintf("web-%d", cmd.At.UnixNano())

	if err := appendControlCommand(s.runtimeDir, cmd); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":     cmd.ID,
		"action": cmd.Action,
		"status": "ok",
		"at":     cmd.At,
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"provider": s.config.Routing.Defaults.Provider,
		"model":    s.config.Routing.Defaults.Model,
		"autonomy": s.config.Autonomy,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleReviewDiff(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	items, err := loop.ReadPendingReview(s.runtimeDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var found *loop.PendingReviewItem
	for i := range items {
		if items[i].ID == id {
			found = &items[i]
			break
		}
	}
	if found == nil {
		http.Error(w, "review not found", http.StatusNotFound)
		return
	}
	result, err := worktree.DiffWorktree(found.WorktreePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func appendControlCommand(runtimeDir string, cmd loop.ControlCommand) error {
	path := filepath.Join(runtimeDir, "control.ndjson")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create control directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open control file: %w", err)
	}
	defer file.Close()

	line, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("encode control command: %w", err)
	}
	if _, err := file.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("append control command: %w", err)
	}
	return nil
}

// uiOrPlaceholder returns a handler that serves the embedded SPA if ui is
// non-nil, falling back to a placeholder HTML page for unmatched paths.
// If ui is nil, all requests get the placeholder.
//
// For SPA client-side routing: "/" serves index.html, known files are served
// directly, and unknown paths fall back to index.html (client-side routing).
//
// index.html is served directly from memory rather than via http.FileServer,
// because FileServer redirects "/index.html" to "/" (causing a loop).
func uiOrPlaceholder(ui fs.FS) http.Handler {
	if ui == nil {
		return http.HandlerFunc(servePlaceholder)
	}

	// Read index.html once at startup.
	indexHTML, err := fs.ReadFile(ui, "index.html")
	if err != nil {
		return http.HandlerFunc(servePlaceholder)
	}

	fileServer := http.FileServer(http.FS(ui))
	serveIndex := func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" || path == "" {
			serveIndex(w)
			return
		}
		// Check if the file exists in the embedded FS.
		f, err := ui.Open(path[1:]) // strip leading /
		if err != nil {
			// SPA fallback: serve index.html for client-side routing.
			serveIndex(w)
			return
		}
		f.Close()
		fileServer.ServeHTTP(w, r)
	})
}

func servePlaceholder(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(placeholderHTML)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

// corsMiddleware allows requests from any localhost origin (dev).
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isLocalhostOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isLocalhostOrigin(origin string) bool {
	origin = strings.ToLower(strings.TrimSpace(origin))
	if origin == "" {
		return false
	}
	return strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:") ||
		strings.HasPrefix(origin, "http://[::1]:")
}

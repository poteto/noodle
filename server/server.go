package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/snapshot"
	"github.com/poteto/noodle/loop"
)

// Options configures the HTTP server.
type Options struct {
	RuntimeDir string
	Addr       string // host:port, defaults to "127.0.0.1:0"
	Now        func() time.Time
}

// Server serves the web UI API.
type Server struct {
	runtimeDir string
	now        func() time.Time
	httpServer *http.Server
	listener   net.Listener
	sse        *sseHub
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

	s := &Server{
		runtimeDir: runtimeDir,
		now:        now,
		sse:        newSSEHub(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/events", s.handleSSE)
	mux.HandleFunc("GET /api/snapshot", s.handleSnapshot)
	mux.HandleFunc("GET /api/sessions/{id}/events", s.handleSessionEvents)
	mux.HandleFunc("POST /api/control", s.handleControl)
	mux.HandleFunc("GET /api/config", s.handleConfig)
	mux.HandleFunc("GET /", handleIndex)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: corsMiddleware(mux),
	}
	return s
}

// Start begins listening and serving. It starts the SSE file watcher in the
// background. Blocks until the server shuts down.
func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s.listener = ln

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

	writeJSON(w, http.StatusOK, map[string]string{
		"id":     cmd.ID,
		"action": cmd.Action,
		"status": "queued",
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	cfg, _, err := config.Load("")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return a subset relevant to the UI.
	resp := map[string]any{
		"autonomy":    cfg.Autonomy,
		"concurrency": cfg.Concurrency.MaxCooks,
		"routing": map[string]any{
			"provider": cfg.Routing.Defaults.Provider,
			"model":    cfg.Routing.Defaults.Model,
		},
	}
	writeJSON(w, http.StatusOK, resp)
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

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html><html><head><title>noodle</title></head><body><p>noodle web ui</p></body></html>`)
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

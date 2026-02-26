package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/internal/snapshot"
	"github.com/poteto/noodle/loop"
)

func testServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	// Create minimal runtime files so LoadSnapshot doesn't fail.
	if err := os.MkdirAll(filepath.Join(dir, "sessions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "queue.json"), []byte(`{"items":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "status.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	fixed := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	s := New(Options{
		RuntimeDir: dir,
		Addr:       "127.0.0.1:0",
		Now:        func() time.Time { return fixed },
	})
	return s, dir
}

func TestGetSnapshot(t *testing.T) {
	s, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/snapshot", nil)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content-type = %q, want application/json", ct)
	}

	var snap snapshot.Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if snap.UpdatedAt.IsZero() {
		t.Fatal("expected non-zero updated_at")
	}
}

func TestGetSessionEvents(t *testing.T) {
	s, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/sessions/nonexistent/events", nil)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var events []snapshot.EventLine
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		t.Fatalf("decode events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("event count = %d, want 0", len(events))
	}
}

func TestPostControl(t *testing.T) {
	s, dir := testServer(t)

	body := strings.NewReader(`{"action":"pause"}`)
	req := httptest.NewRequest("POST", "/api/control", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var ack map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&ack); err != nil {
		t.Fatalf("decode ack: %v", err)
	}
	if ack["action"] != "pause" {
		t.Fatalf("action = %q, want pause", ack["action"])
	}
	if ack["status"] != "ok" {
		t.Fatalf("status = %q, want ok", ack["status"])
	}
	if ack["id"] == "" {
		t.Fatal("expected non-empty id")
	}

	// Verify the command was written.
	data, err := os.ReadFile(filepath.Join(dir, "control.ndjson"))
	if err != nil {
		t.Fatalf("read control.ndjson: %v", err)
	}
	var cmd loop.ControlCommand
	if err := json.Unmarshal(data[:len(data)-1], &cmd); err != nil {
		t.Fatalf("parse command: %v", err)
	}
	if cmd.Action != "pause" {
		t.Fatalf("action = %q, want pause", cmd.Action)
	}
}

func TestPostControlMissingAction(t *testing.T) {
	s, _ := testServer(t)

	body := strings.NewReader(`{"order_id":"123"}`)
	req := httptest.NewRequest("POST", "/api/control", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Result().StatusCode)
	}
}

func TestPostControlInvalidJSON(t *testing.T) {
	s, _ := testServer(t)

	body := strings.NewReader(`not json`)
	req := httptest.NewRequest("POST", "/api/control", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Result().StatusCode)
	}
}

func TestCORSHeaders(t *testing.T) {
	s, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/snapshot", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	resp := w.Result()
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("CORS origin = %q, want http://localhost:5173", got)
	}
}

func TestCORSPreflight(t *testing.T) {
	s, _ := testServer(t)

	req := httptest.NewRequest("OPTIONS", "/api/control", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") {
		t.Fatalf("allowed methods = %q, want POST in it", got)
	}
}

func TestCORSBlocksNonLocalhost(t *testing.T) {
	s, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/snapshot", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	resp := w.Result()
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS header for non-localhost, got %q", got)
	}
}

func TestSSEStream(t *testing.T) {
	s, _ := testServer(t)

	// Use a real HTTP test server for SSE.
	ts := httptest.NewServer(s.httpServer.Handler)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("SSE request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content-type = %q, want text/event-stream", ct)
	}

	// Read the initial snapshot event (unnamed SSE — just data: lines).
	scanner := bufio.NewScanner(resp.Body)
	var dataLine string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			dataLine = strings.TrimPrefix(line, "data: ")
			break
		}
	}

	if dataLine == "" {
		t.Fatal("expected non-empty data line")
	}

	var snap snapshot.Snapshot
	if err := json.Unmarshal([]byte(dataLine), &snap); err != nil {
		t.Fatalf("decode SSE snapshot: %v", err)
	}
	if snap.UpdatedAt.IsZero() {
		t.Fatal("expected non-zero updated_at in SSE snapshot")
	}
}

func TestIsLocalhostOrigin(t *testing.T) {
	cases := []struct {
		origin string
		want   bool
	}{
		{"http://localhost:5173", true},
		{"http://127.0.0.1:5173", true},
		{"http://[::1]:5173", true},
		{"https://evil.com", false},
		{"http://localhost.evil.com:5173", false},
		{"", false},
	}
	for _, tc := range cases {
		got := isLocalhostOrigin(tc.origin)
		if got != tc.want {
			t.Errorf("isLocalhostOrigin(%q) = %v, want %v", tc.origin, got, tc.want)
		}
	}
}

func TestPostControlEnqueue(t *testing.T) {
	s, dir := testServer(t)

	body := strings.NewReader(`{"action":"enqueue","order_id":"task-99","prompt":"Fix the bug","task_key":"execute","provider":"claude","model":"claude-opus-4-6"}`)
	req := httptest.NewRequest("POST", "/api/control", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}

	data, err := os.ReadFile(filepath.Join(dir, "control.ndjson"))
	if err != nil {
		t.Fatalf("read control: %v", err)
	}
	var cmd loop.ControlCommand
	if err := json.Unmarshal(data[:len(data)-1], &cmd); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cmd.Action != "enqueue" {
		t.Fatalf("action = %q", cmd.Action)
	}
	if cmd.Prompt != "Fix the bug" {
		t.Fatalf("prompt = %q", cmd.Prompt)
	}
	if cmd.TaskKey != "execute" {
		t.Fatalf("task_key = %q", cmd.TaskKey)
	}
}

func TestSSEHubDiffGating(t *testing.T) {
	hub := newSSEHub()
	dir := t.TempDir()

	// Create minimal runtime files.
	os.MkdirAll(filepath.Join(dir, "sessions"), 0o755)
	os.WriteFile(filepath.Join(dir, "queue.json"), []byte(`{"items":[]}`), 0o644)
	os.WriteFile(filepath.Join(dir, "status.json"), []byte(`{}`), 0o644)

	fixed := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	now := func() time.Time { return fixed }

	client := &sseClient{
		ch:     make(chan []byte, 16),
		closed: make(chan struct{}),
	}
	hub.addClient(client)

	// First load should broadcast.
	hub.loadAndBroadcast(dir, now)
	select {
	case msg := <-client.ch:
		if !strings.Contains(string(msg), "data: ") {
			t.Fatalf("expected data event, got %s", msg)
		}
	default:
		t.Fatal("expected broadcast on first load")
	}

	// Second load with same data should NOT broadcast (diff gating).
	hub.loadAndBroadcast(dir, now)
	select {
	case msg := <-client.ch:
		t.Fatalf("expected no broadcast on unchanged data, got %s", msg)
	default:
		// Good, no message.
	}

	hub.removeClient(client)
}

func TestGetIndex(t *testing.T) {
	s, _ := testServer(t)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("content-type = %q, want text/html", ct)
	}
}

func TestFindPort(t *testing.T) {
	addr, err := FindPort(0)
	if err != nil {
		t.Fatalf("FindPort(0): %v", err)
	}
	if addr == "" {
		t.Fatal("expected non-empty addr")
	}
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Fatalf("addr = %q, want 127.0.0.1:*", addr)
	}
}

func TestFindPortSkipsBusy(t *testing.T) {
	// Occupy a port, then verify FindPort skips it.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	addr, err := FindPort(port)
	if err != nil {
		t.Fatalf("FindPort(%d): %v", port, err)
	}
	// Should have found port+1 or later.
	if addr == fmt.Sprintf("127.0.0.1:%d", port) {
		t.Fatalf("FindPort returned the busy port %d", port)
	}
}

func TestSessionEventsAfterFilter(t *testing.T) {
	s, _ := testServer(t)

	// No sessions exist, so events for any session should be empty.
	req := httptest.NewRequest("GET", "/api/sessions/cook-a/events?after=2026-02-25T00:00:00Z", nil)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var events []snapshot.EventLine
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		t.Fatalf("decode events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("event count = %d, want 0", len(events))
	}
}

package server

import (
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

	"github.com/gorilla/websocket"
	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/internal/snapshot"
	"github.com/poteto/noodle/loop"
)

type staticProvider struct {
	state loop.LoopState
}

func (p *staticProvider) State() loop.LoopState { return p.state }

func testServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()

	broker := NewSessionEventBroker()
	fixed := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	s := New(Options{
		RuntimeDir:        dir,
		Addr:              "127.0.0.1:0",
		Now:               func() time.Time { return fixed },
		LoopStateProvider: &staticProvider{state: loop.LoopState{Status: "running"}},
		Broker:            broker,
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

func TestGetSessionEventsMissingSessionID(t *testing.T) {
	s, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/sessions/missing/events", nil)
	req.SetPathValue("id", " ")
	w := httptest.NewRecorder()
	s.handleSessionEvents(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	message := strings.TrimSpace(string(body))
	if message != "session ID missing" {
		t.Fatalf("message = %q, want %q", message, "session ID missing")
	}
	assertFailureStateMessage(t, message)
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

	var ack loop.ControlAck
	if err := json.NewDecoder(resp.Body).Decode(&ack); err != nil {
		t.Fatalf("decode ack: %v", err)
	}
	if ack.Action != "pause" {
		t.Fatalf("action = %q, want pause", ack.Action)
	}
	if ack.Status != "ok" {
		t.Fatalf("status = %q, want ok", ack.Status)
	}
	if ack.ID == "" {
		t.Fatal("expected non-empty id")
	}
	if ack.Failure != nil {
		t.Fatalf("ack failure = %#v, want nil", ack.Failure)
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

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	message := strings.TrimSpace(string(rawBody))
	if message != "control action missing" {
		t.Fatalf("message = %q, want %q", message, "control action missing")
	}
	assertFailureStateMessage(t, message)
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

func TestWSConnection(t *testing.T) {
	s, _ := testServer(t)

	ts := httptest.NewServer(s.httpServer.Handler)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WS dial: %v", err)
	}
	defer conn.Close()

	// Read the initial snapshot.
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read initial message: %v", err)
	}

	var envelope struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(msg, &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if envelope.Type != "snapshot" {
		t.Fatalf("type = %q, want snapshot", envelope.Type)
	}

	var snap snapshot.Snapshot
	if err := json.Unmarshal(envelope.Data, &snap); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if snap.UpdatedAt.IsZero() {
		t.Fatal("expected non-zero updated_at in WS snapshot")
	}
}

func TestWSHubDiffGating(t *testing.T) {
	broker := NewSessionEventBroker()
	hub := newWSHub(broker)
	dir := t.TempDir()

	fixed := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	now := func() time.Time { return fixed }

	provider := &staticProvider{state: loop.LoopState{Status: "running"}}

	sub := &mockSubscriber{}
	// Wrap mock as a wsClient-like subscriber via the hub's broadcast path.
	// Instead, test directly via loadAndBroadcast by adding a real wsClient.
	// Use a channel-based approach: add a fake wsClient with a buffered send channel.
	fakeSend := make(chan []byte, 16)
	fakeClient := &wsClient{
		send: fakeSend,
		hub:  hub,
	}
	hub.addClient(fakeClient)
	_ = sub // not used in this approach

	// First load should broadcast.
	hub.loadAndBroadcast(dir, now, provider, nil)
	select {
	case msg := <-fakeSend:
		if !strings.Contains(string(msg), `"type":"snapshot"`) {
			t.Fatalf("expected snapshot envelope, got %s", msg)
		}
	default:
		t.Fatal("expected broadcast on first load")
	}

	// Second load with same data should NOT broadcast (diff gating).
	hub.loadAndBroadcast(dir, now, provider, nil)
	select {
	case msg := <-fakeSend:
		t.Fatalf("expected no broadcast on unchanged data, got %s", msg)
	default:
		// Good, no message.
	}

	hub.removeClient(fakeClient)
}

func TestWSSubscribeSessionEvents(t *testing.T) {
	s, dir := testServer(t)

	// Write some events to disk for backfill.
	sessionID := "test-session-1"
	sessionDir := filepath.Join(dir, "sessions", sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir session dir: %v", err)
	}
	ev := event.Event{
		Type:      event.EventAction,
		Payload:   json.RawMessage(`{"tool":"read","message":"reading file"}`),
		Timestamp: time.Now(),
		SessionID: sessionID,
	}
	evData, _ := json.Marshal(ev)
	if err := os.WriteFile(filepath.Join(sessionDir, "events.ndjson"), append(evData, '\n'), 0o644); err != nil {
		t.Fatalf("write events: %v", err)
	}

	ts := httptest.NewServer(s.httpServer.Handler)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WS dial: %v", err)
	}
	defer conn.Close()

	// Read and discard the initial snapshot.
	conn.ReadMessage()

	// Send subscribe.
	subMsg, _ := json.Marshal(map[string]string{
		"type":       "subscribe",
		"session_id": sessionID,
	})
	if err := conn.WriteMessage(websocket.TextMessage, subMsg); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}

	// Read backfill.
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read backfill: %v", err)
	}
	var backfill struct {
		Type      string `json:"type"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(raw, &backfill); err != nil {
		t.Fatalf("unmarshal backfill: %v", err)
	}
	if backfill.Type != "backfill" {
		t.Fatalf("type = %q, want backfill", backfill.Type)
	}
	if backfill.SessionID != sessionID {
		t.Fatalf("session_id = %q, want %q", backfill.SessionID, sessionID)
	}

	// Read subscribed ack.
	_, raw, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("read subscribed: %v", err)
	}
	var ack struct {
		Type      string `json:"type"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(raw, &ack); err != nil {
		t.Fatalf("unmarshal ack: %v", err)
	}
	if ack.Type != "subscribed" {
		t.Fatalf("type = %q, want subscribed", ack.Type)
	}
}

func TestWSControl(t *testing.T) {
	s, dir := testServer(t)

	ts := httptest.NewServer(s.httpServer.Handler)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WS dial: %v", err)
	}
	defer conn.Close()

	// Read and discard the initial snapshot.
	conn.ReadMessage()

	// Send control command via WS.
	ctrlMsg, _ := json.Marshal(map[string]any{
		"type": "control",
		"data": map[string]string{
			"action": "pause",
		},
	})
	if err := conn.WriteMessage(websocket.TextMessage, ctrlMsg); err != nil {
		t.Fatalf("write control: %v", err)
	}

	// Read control ack.
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read control ack: %v", err)
	}
	var ackEnv struct {
		Type string          `json:"type"`
		Data loop.ControlAck `json:"data"`
	}
	if err := json.Unmarshal(raw, &ackEnv); err != nil {
		t.Fatalf("unmarshal ack: %v", err)
	}
	if ackEnv.Type != "control_ack" {
		t.Fatalf("type = %q, want control_ack", ackEnv.Type)
	}
	if ackEnv.Data.Action != "pause" {
		t.Fatalf("action = %v, want pause", ackEnv.Data.Action)
	}
	if ackEnv.Data.Failure != nil {
		t.Fatalf("ack failure = %#v, want nil", ackEnv.Data.Failure)
	}

	// Verify the command was written to disk.
	data, err := os.ReadFile(filepath.Join(dir, "control.ndjson"))
	if err != nil {
		t.Fatalf("read control.ndjson: %v", err)
	}
	if !strings.Contains(string(data), "pause") {
		t.Fatalf("control.ndjson missing pause command")
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

func TestSnapshotIncludesWarnings(t *testing.T) {
	dir := t.TempDir()
	fixed := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)

	s := New(Options{
		RuntimeDir:        dir,
		Addr:              "127.0.0.1:0",
		Now:               func() time.Time { return fixed },
		LoopStateProvider: &staticProvider{state: loop.LoopState{Status: "running"}},
		Warnings:          []string{"unknown runtime \"tmux\" (valid: process, sprites, cursor)"},
	})

	req := httptest.NewRequest("GET", "/api/snapshot", nil)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var snap snapshot.Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if len(snap.Warnings) != 1 {
		t.Fatalf("warnings count = %d, want 1", len(snap.Warnings))
	}
	if snap.Warnings[0] != "unknown runtime \"tmux\" (valid: process, sprites, cursor)" {
		t.Fatalf("warning = %q", snap.Warnings[0])
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

func assertFailureStateMessage(t *testing.T, message string) {
	t.Helper()
	lower := strings.ToLower(message)
	for _, term := range []string{"must", "required", "requires", "expected"} {
		if strings.Contains(lower, term) {
			t.Fatalf("message %q contains expectation-style term %q", message, term)
		}
	}
}

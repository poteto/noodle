package dispatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestClaudeControllerSteerableReturnsTrue(t *testing.T) {
	ctrl := newClaudeController(&bytes.Buffer{})
	if !ctrl.Steerable() {
		t.Fatal("Steerable() = false, want true")
	}
}

func TestClaudeControllerSendMessageWritesJSONWhenIdle(t *testing.T) {
	var buf bytes.Buffer
	ctrl := newClaudeController(&buf)

	if err := ctrl.SendMessage(context.Background(), "hello world"); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	line := strings.TrimSpace(buf.String())
	var msg streamJSONUserMessage
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Type != "user" {
		t.Fatalf("type = %q, want user", msg.Type)
	}
	if msg.Message.Role != "user" {
		t.Fatalf("role = %q, want user", msg.Message.Role)
	}
	if msg.Message.Content != "hello world" {
		t.Fatalf("content = %q, want %q", msg.Message.Content, "hello world")
	}
}

func TestClaudeControllerSendMessageWritesNewlineTerminated(t *testing.T) {
	var buf bytes.Buffer
	ctrl := newClaudeController(&buf)

	if err := ctrl.SendMessage(context.Background(), "test"); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	if !strings.HasSuffix(buf.String(), "\n") {
		t.Fatal("message not newline-terminated")
	}
}

func TestClaudeControllerSendMessageTransitionsToActive(t *testing.T) {
	ctrl := newClaudeController(&bytes.Buffer{})

	if err := ctrl.SendMessage(context.Background(), "go"); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	ctrl.mu.Lock()
	state := ctrl.state
	ctrl.mu.Unlock()

	if state != turnActive {
		t.Fatalf("state = %d, want turnActive (%d)", state, turnActive)
	}
}

func TestClaudeControllerInterruptWritesControlRequest(t *testing.T) {
	var buf bytes.Buffer
	ctrl := newClaudeController(&buf)

	// Put controller into active state.
	ctrl.mu.Lock()
	ctrl.state = turnActive
	ctrl.turnDone = make(chan struct{})
	done := ctrl.turnDone
	ctrl.mu.Unlock()

	// Simulate result arriving shortly after interrupt.
	go func() {
		time.Sleep(50 * time.Millisecond)
		close(done)
		ctrl.mu.Lock()
		ctrl.state = turnIdle
		ctrl.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ctrl.Interrupt(ctx); err != nil {
		t.Fatalf("Interrupt: %v", err)
	}

	line := strings.TrimSpace(buf.String())
	var msg streamJSONControlRequest
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Type != "control_request" {
		t.Fatalf("type = %q, want control_request", msg.Type)
	}
	if msg.Request.Subtype != "interrupt" {
		t.Fatalf("subtype = %q, want interrupt", msg.Request.Subtype)
	}
	if msg.RequestID == "" {
		t.Fatal("request_id is empty")
	}
}

func TestClaudeControllerInterruptTimesOut(t *testing.T) {
	ctrl := newClaudeController(&bytes.Buffer{})

	// Put controller into active state with a turnDone that never closes.
	ctrl.mu.Lock()
	ctrl.state = turnActive
	ctrl.turnDone = make(chan struct{})
	ctrl.mu.Unlock()

	// Use a very short context deadline to test timeout behavior
	// without waiting 30 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := ctrl.Interrupt(ctx)
	if err == nil {
		t.Fatal("Interrupt should have returned error on timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
}

func TestClaudeControllerInterruptWhenIdleIsNoop(t *testing.T) {
	ctrl := newClaudeController(&bytes.Buffer{})

	if err := ctrl.Interrupt(context.Background()); err != nil {
		t.Fatalf("Interrupt on idle controller: %v", err)
	}
}

func TestClaudeControllerNotifyEventInitTransitionsToActive(t *testing.T) {
	ctrl := newClaudeController(&bytes.Buffer{})

	ctrl.NotifyEvent("init")

	ctrl.mu.Lock()
	state := ctrl.state
	ctrl.mu.Unlock()

	if state != turnActive {
		t.Fatalf("state = %d after init, want turnActive (%d)", state, turnActive)
	}
}

func TestClaudeControllerNotifyEventResultTransitionsToIdle(t *testing.T) {
	ctrl := newClaudeController(&bytes.Buffer{})

	// Start active.
	ctrl.mu.Lock()
	ctrl.state = turnActive
	ctrl.turnDone = make(chan struct{})
	ctrl.mu.Unlock()

	ctrl.NotifyEvent("result")

	ctrl.mu.Lock()
	state := ctrl.state
	ctrl.mu.Unlock()

	if state != turnIdle {
		t.Fatalf("state = %d after result, want turnIdle (%d)", state, turnIdle)
	}
}

func TestClaudeControllerSendMessageQueuesWhenActive(t *testing.T) {
	var buf bytes.Buffer
	ctrl := newClaudeController(&buf)

	// Send first message to become active.
	if err := ctrl.SendMessage(context.Background(), "first"); err != nil {
		t.Fatalf("first SendMessage: %v", err)
	}
	buf.Reset()

	// Send second message in background — it should block.
	done := make(chan error, 1)
	go func() {
		done <- ctrl.SendMessage(context.Background(), "second")
	}()

	// Give the goroutine time to block.
	time.Sleep(50 * time.Millisecond)

	// No output yet — second message is queued.
	if buf.Len() > 0 {
		t.Fatal("second message written before turn completed")
	}

	// Simulate turn completion.
	ctrl.NotifyEvent("result")

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("second SendMessage: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("second SendMessage did not unblock")
	}

	// Verify the second message was written.
	line := strings.TrimSpace(buf.String())
	var msg streamJSONUserMessage
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Message.Content != "second" {
		t.Fatalf("content = %q, want second", msg.Message.Content)
	}
}

func TestClaudeControllerSerializesConcurrentOperations(t *testing.T) {
	var buf safeBuffer
	ctrl := newClaudeController(&buf)

	// Send first message to become active.
	if err := ctrl.SendMessage(context.Background(), "first"); err != nil {
		t.Fatalf("first SendMessage: %v", err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)

	// Two concurrent steers: one sends, one interrupts.
	wg.Add(2)
	go func() {
		defer wg.Done()
		errs <- ctrl.SendMessage(context.Background(), "concurrent-send")
	}()
	go func() {
		defer wg.Done()
		errs <- ctrl.Interrupt(context.Background())
	}()

	// Let them queue up, then complete the turn.
	time.Sleep(50 * time.Millisecond)
	ctrl.NotifyEvent("result")

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent operation error: %v", err)
		}
	}
}

func TestProcessSessionControllerReturnsNoopWithoutController(t *testing.T) {
	session := &processSession{}
	ctrl := session.Controller()
	if ctrl.Steerable() {
		t.Fatal("Controller().Steerable() = true without controller, want false")
	}
	if !errors.Is(ctrl.SendMessage(context.Background(), "x"), ErrNotSteerable) {
		t.Fatal("Controller().SendMessage should return ErrNotSteerable")
	}
}

func TestProcessSessionControllerReturnsClaudeController(t *testing.T) {
	cc := newClaudeController(&bytes.Buffer{})
	session := &processSession{controller: cc}
	ctrl := session.Controller()
	if !ctrl.Steerable() {
		t.Fatal("Controller().Steerable() = false with claude controller, want true")
	}
}

func TestReplaceFlagRemovesOldAndAppendsNew(t *testing.T) {
	args := []string{"-p", "--output-format", "stream-json", "--verbose"}
	got := replaceFlag(args, "-p", "--input-format", "stream-json")

	for _, a := range got {
		if a == "-p" {
			t.Fatal("replaceFlag did not remove -p")
		}
	}

	found := false
	for i, a := range got {
		if a == "--input-format" && i+1 < len(got) && got[i+1] == "stream-json" {
			found = true
		}
	}
	if !found {
		t.Fatalf("replaceFlag did not add --input-format stream-json: %v", got)
	}
}

// safeBuffer is a concurrency-safe bytes.Buffer for tests.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

package loop

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	loopruntime "github.com/poteto/noodle/runtime"
)

// mockController records calls and allows configurable behavior.
type mockController struct {
	mu              sync.Mutex
	steerable       bool
	interruptErr    error
	sendMessageErr  error
	interruptCalls  int
	sendCalls       int
	lastSentMessage string
}

func (c *mockController) Steerable() bool { return c.steerable }

func (c *mockController) Interrupt(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.interruptCalls++
	return c.interruptErr
}

func (c *mockController) SendMessage(_ context.Context, prompt string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sendCalls++
	c.lastSentMessage = prompt
	return c.sendMessageErr
}

// steerableSession wraps a mockSession to return a controllable controller.
type steerableSession struct {
	*mockSession
	ctrl loopruntime.AgentController
}

func (s *steerableSession) Controller() loopruntime.AgentController { return s.ctrl }

func newSteerTestLoop(t *testing.T, rt *mockRuntime) *Loop {
	t.Helper()
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	return New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})
}

func TestSteerLiveInterruptAndRedirect(t *testing.T) {
	rt := newMockRuntime()
	l := newSteerTestLoop(t, rt)

	ctrl := &mockController{steerable: true}
	sess := &steerableSession{
		mockSession: &mockSession{id: "sess-live", status: "running", done: make(chan struct{})},
		ctrl:        ctrl,
	}
	l.cooks.activeCooksByOrder["order-1"] = &cookHandle{
		cookIdentity: cookIdentity{orderID: "order-1"},
		session:      sess,
		worktreeName: "wt-1",
	}

	if err := l.steer("wt-1", "change direction"); err != nil {
		t.Fatalf("steer: %v", err)
	}

	// steerLive runs async — wait for it.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		ctrl.mu.Lock()
		sent := ctrl.sendCalls
		ctrl.mu.Unlock()
		if sent > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()

	if ctrl.interruptCalls != 1 {
		t.Fatalf("interrupt calls = %d, want 1", ctrl.interruptCalls)
	}
	if ctrl.sendCalls != 1 {
		t.Fatalf("send calls = %d, want 1", ctrl.sendCalls)
	}
	if ctrl.lastSentMessage != "change direction" {
		t.Fatalf("sent message = %q, want %q", ctrl.lastSentMessage, "change direction")
	}

	// Session should NOT have been killed — still running.
	if sess.status == "killed" {
		t.Fatal("steerable session was killed — should have been redirected")
	}
	// Cook should still be in the active map.
	if _, ok := l.cooks.activeCooksByOrder["order-1"]; !ok {
		t.Fatal("cook was removed from active map — should remain for live steer")
	}
}

func TestSteerNonSteerableFallsBackToRespawn(t *testing.T) {
	rt := newMockRuntime()
	l := newSteerTestLoop(t, rt)

	sess := &mockSession{id: "sess-noop", status: "running", done: make(chan struct{})}
	l.cooks.activeCooksByOrder["order-2"] = &cookHandle{
		cookIdentity: cookIdentity{
			orderID: "order-2",
			stage:   Stage{TaskKey: "execute", Prompt: "original", Provider: "claude", Model: "claude-sonnet-4-6"},
		},
		session:      sess,
		worktreeName: "wt-2",
		worktreePath: t.TempDir(),
		orderStatus:  OrderStatusActive,
	}

	err := l.steer("wt-2", "new direction")
	if err != nil {
		t.Fatalf("steer: %v", err)
	}

	// Original session should be killed.
	if sess.status != "killed" {
		t.Fatalf("session status = %q, want killed", sess.status)
	}
	// spawnCook re-adds the order with a new session — verify the session changed.
	newCook, ok := l.cooks.activeCooksByOrder["order-2"]
	if !ok {
		t.Fatal("respawned cook not in active map")
	}
	if newCook.session.ID() == "sess-noop" {
		t.Fatal("session was not respawned — same session ID")
	}
}

func TestSteerLiveInterruptFailureLogs(t *testing.T) {
	rt := newMockRuntime()
	l := newSteerTestLoop(t, rt)

	ctrl := &mockController{
		steerable:    true,
		interruptErr: errors.New("interrupt timeout"),
	}
	sess := &steerableSession{
		mockSession: &mockSession{id: "sess-fail", status: "running", done: make(chan struct{})},
		ctrl:        ctrl,
	}
	l.cooks.activeCooksByOrder["order-3"] = &cookHandle{
		cookIdentity: cookIdentity{orderID: "order-3"},
		session:      sess,
		worktreeName: "wt-3",
	}

	// steer returns nil (async), but the goroutine should handle the failure.
	if err := l.steer("wt-3", "redirect"); err != nil {
		t.Fatalf("steer: %v", err)
	}

	// Wait for async steer to complete.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		ctrl.mu.Lock()
		calls := ctrl.interruptCalls
		ctrl.mu.Unlock()
		if calls > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()

	if ctrl.interruptCalls != 1 {
		t.Fatalf("interrupt calls = %d, want 1", ctrl.interruptCalls)
	}
	// SendMessage should NOT be called if interrupt failed.
	if ctrl.sendCalls != 0 {
		t.Fatalf("send calls = %d, want 0 (interrupt failed)", ctrl.sendCalls)
	}
}

func TestSteerSessionNotFound(t *testing.T) {
	rt := newMockRuntime()
	l := newSteerTestLoop(t, rt)

	err := l.steer("nonexistent", "hello")
	if err == nil {
		t.Fatal("expected error for missing session")
	}
	if err.Error() != "session not found" {
		t.Fatalf("error = %q, want 'session not found'", err.Error())
	}
}

func TestSteerScheduleTargetsActiveScheduleSession(t *testing.T) {
	rt := newMockRuntime()
	l := newSteerTestLoop(t, rt)

	ctrl := &mockController{steerable: true}
	sess := &steerableSession{
		mockSession: &mockSession{id: "sess-schedule", status: "running", done: make(chan struct{})},
		ctrl:        ctrl,
	}
	l.cooks.activeCooksByOrder[ScheduleTaskKey()] = &cookHandle{
		cookIdentity: cookIdentity{
			orderID: ScheduleTaskKey(),
			stage: Stage{
				TaskKey: ScheduleTaskKey(),
			},
		},
		session: sess,
	}

	if err := l.steer(ScheduleTaskKey(), "focus on auth bugs"); err != nil {
		t.Fatalf("steer: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		ctrl.mu.Lock()
		sent := ctrl.sendCalls
		ctrl.mu.Unlock()
		if sent > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()

	if ctrl.interruptCalls != 1 {
		t.Fatalf("interrupt calls = %d, want 1", ctrl.interruptCalls)
	}
	if ctrl.sendCalls != 1 {
		t.Fatalf("send calls = %d, want 1", ctrl.sendCalls)
	}
	if ctrl.lastSentMessage != "focus on auth bugs" {
		t.Fatalf("sent message = %q, want %q", ctrl.lastSentMessage, "focus on auth bugs")
	}
}

func TestSteerSerializesConcurrentSteers(t *testing.T) {
	rt := newMockRuntime()
	l := newSteerTestLoop(t, rt)

	var callOrder atomic.Int64
	var firstSendStart, secondSendStart atomic.Int64

	ctrl := &mockController{steerable: true}
	// Override SendMessage to track ordering.
	slowCtrl := &serializedController{
		mockController: ctrl,
		sendDelay:      50 * time.Millisecond,
		callOrder:      &callOrder,
		firstStart:     &firstSendStart,
		secondStart:    &secondSendStart,
	}

	sess := &steerableSession{
		mockSession: &mockSession{id: "sess-serial", status: "running", done: make(chan struct{})},
		ctrl:        slowCtrl,
	}
	l.cooks.activeCooksByOrder["order-s"] = &cookHandle{
		cookIdentity: cookIdentity{orderID: "order-s"},
		session:      sess,
		worktreeName: "wt-serial",
	}

	// Launch two steers concurrently.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = l.steer("wt-serial", "first")
	}()
	go func() {
		defer wg.Done()
		_ = l.steer("wt-serial", "second")
	}()
	wg.Wait()

	// Wait for both async operations.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		slowCtrl.mu.Lock()
		sends := slowCtrl.sendCalls
		slowCtrl.mu.Unlock()
		if sends >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Both steers should have completed.
	slowCtrl.mu.Lock()
	sends := slowCtrl.sendCalls
	slowCtrl.mu.Unlock()
	if sends != 2 {
		t.Fatalf("send calls = %d, want 2", sends)
	}

	// Verify serialization: second send should start after first finishes.
	first := firstSendStart.Load()
	second := secondSendStart.Load()
	if first == 0 || second == 0 {
		t.Fatal("expected both sends to have recorded timestamps")
	}
	// The sends should be separated by at least the delay.
	diff := second - first
	if diff < 0 {
		diff = -diff
	}
	if diff < int64(40*time.Millisecond) {
		t.Fatalf("sends overlapped: diff=%v, want >= 40ms", time.Duration(diff))
	}
}

// serializedController wraps mockController with a delay to test serialization.
type serializedController struct {
	*mockController
	sendDelay   time.Duration
	callOrder   *atomic.Int64
	firstStart  *atomic.Int64
	secondStart *atomic.Int64
}

func (c *serializedController) SendMessage(ctx context.Context, prompt string) error {
	seq := c.callOrder.Add(1)
	now := time.Now().UnixNano()
	if seq == 1 {
		c.firstStart.Store(now)
	} else {
		c.secondStart.Store(now)
	}
	time.Sleep(c.sendDelay)

	c.mu.Lock()
	defer c.mu.Unlock()
	c.sendCalls++
	c.lastSentMessage = prompt
	return c.sendMessageErr
}

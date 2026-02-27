package dispatcher

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// claudeInterruptTimeout is the maximum time to wait for Claude to acknowledge
// an interrupt before returning an error.
const claudeInterruptTimeout = 30 * time.Second

// turnState represents the current turn state of a Claude session.
type turnState int

const (
	turnIdle         turnState = iota
	turnActive                         // agent is processing a turn
	turnInterrupting                   // interrupt sent, waiting for result
)

// claudeController implements AgentController for Claude Code sessions using
// the --input-format stream-json / --output-format stream-json protocol.
//
// Turn state is tracked via NotifyEvent calls from processSession's event
// interceptor — the controller never reads stdout itself.
type claudeController struct {
	mu    sync.Mutex
	state turnState
	w     io.Writer // stdin pipe of the child process

	// turnDone is signalled when a turn completes (result event received).
	// Recreated each time a turn starts.
	turnDone chan struct{}

	// pendingPrompt holds a queued message to send after the current turn.
	pendingPrompt string
}

// newClaudeController creates a controller wrapping the process's stdin writer.
func newClaudeController(stdin io.Writer) *claudeController {
	return &claudeController{
		w:        stdin,
		state:    turnIdle,
		turnDone: make(chan struct{}),
	}
}

func (c *claudeController) Steerable() bool { return true }

// SendMessage sends a user message to the Claude session. If the agent is idle,
// the message is written immediately. If a turn is active, the message is
// queued and delivered after the current turn completes (or after an interrupt).
func (c *claudeController) SendMessage(ctx context.Context, prompt string) error {
	c.mu.Lock()

	switch c.state {
	case turnIdle:
		err := c.writeUserMessage(prompt)
		if err == nil {
			c.state = turnActive
			c.turnDone = make(chan struct{})
		}
		c.mu.Unlock()
		return err

	case turnActive, turnInterrupting:
		// Queue the message; it will be sent when the turn completes.
		c.pendingPrompt = prompt
		done := c.turnDone
		c.mu.Unlock()

		// Wait for the current turn to finish.
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}

		// Turn completed — send the queued message.
		c.mu.Lock()
		// Another goroutine may have already consumed the pending prompt
		// via drainPending in NotifyEvent; only send if still ours.
		if c.pendingPrompt != prompt {
			c.mu.Unlock()
			return nil
		}
		c.pendingPrompt = ""
		err := c.writeUserMessage(prompt)
		if err == nil {
			c.state = turnActive
			c.turnDone = make(chan struct{})
		}
		c.mu.Unlock()
		return err

	default:
		c.mu.Unlock()
		return fmt.Errorf("unknown turn state")
	}
}

// Interrupt sends a control_request interrupt to Claude and waits for the
// current turn to stop. Returns an error on timeout — caller decides whether
// to fall back to Kill.
func (c *claudeController) Interrupt(ctx context.Context) error {
	c.mu.Lock()

	if c.state == turnIdle {
		c.mu.Unlock()
		return nil
	}

	done := c.turnDone

	if c.state == turnActive {
		c.state = turnInterrupting
		if err := c.writeInterrupt(); err != nil {
			c.state = turnActive
			c.mu.Unlock()
			return fmt.Errorf("write interrupt: %w", err)
		}
	}
	// If already interrupting, just wait on the existing done channel.
	c.mu.Unlock()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(claudeInterruptTimeout):
		return fmt.Errorf("interrupt timed out after %s", claudeInterruptTimeout)
	}
}

// NotifyEvent is called by processSession when it sees init or result events
// in the canonical event stream. This drives the controller's state machine
// without consuming stdout.
func (c *claudeController) NotifyEvent(eventType string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch eventType {
	case "init":
		if c.state == turnIdle {
			c.state = turnActive
			c.turnDone = make(chan struct{})
		}

	case "result":
		if c.state == turnActive || c.state == turnInterrupting {
			c.state = turnIdle
			close(c.turnDone)
			c.drainPending()
		}
	}
}

// drainPending sends any queued message. Must be called with c.mu held.
func (c *claudeController) drainPending() {
	if c.pendingPrompt == "" {
		return
	}
	prompt := c.pendingPrompt
	c.pendingPrompt = ""
	if err := c.writeUserMessage(prompt); err == nil {
		c.state = turnActive
		c.turnDone = make(chan struct{})
	}
}

// writeUserMessage writes a stream-json user message to stdin. Must be called
// with c.mu held.
func (c *claudeController) writeUserMessage(prompt string) error {
	msg := streamJSONUserMessage{
		Type: "user",
		Message: streamJSONMessageContent{
			Role:    "user",
			Content: prompt,
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("encode user message: %w", err)
	}
	data = append(data, '\n')
	_, err = c.w.Write(data)
	if err != nil {
		return fmt.Errorf("write user message: %w", err)
	}
	return nil
}

// writeInterrupt writes a stream-json control_request interrupt to stdin.
// Must be called with c.mu held.
func (c *claudeController) writeInterrupt() error {
	msg := streamJSONControlRequest{
		Type:      "control_request",
		RequestID: randomRequestID(),
		Request: streamJSONInterruptRequest{
			Subtype: "interrupt",
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("encode interrupt: %w", err)
	}
	data = append(data, '\n')
	_, err = c.w.Write(data)
	if err != nil {
		return fmt.Errorf("write interrupt: %w", err)
	}
	return nil
}

// stream-json wire types.

type streamJSONUserMessage struct {
	Type    string                   `json:"type"`
	Message streamJSONMessageContent `json:"message"`
}

type streamJSONMessageContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type streamJSONControlRequest struct {
	Type      string                      `json:"type"`
	RequestID string                      `json:"request_id"`
	Request   streamJSONInterruptRequest  `json:"request"`
}

type streamJSONInterruptRequest struct {
	Subtype string `json:"subtype"`
}

// randomRequestID returns a random hex string suitable for control_request IDs.
func randomRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

var _ AgentController = (*claudeController)(nil)

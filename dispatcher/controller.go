package dispatcher

import (
	"context"
	"errors"
)

// ErrNotSteerable is returned when SendMessage or Interrupt is called on a
// controller that does not support live steering.
var ErrNotSteerable = errors.New("session not steerable")

// AgentController provides live communication with a running agent session.
// Implementations are provider-specific (Claude pipe transport, Codex
// app-server). Sessions that don't support steering return a noopController.
type AgentController interface {
	// SendMessage sends a user message to the agent. Blocks until delivered
	// (not until the turn completes).
	SendMessage(ctx context.Context, prompt string) error

	// Interrupt requests the agent to stop its current turn. Blocks until
	// acknowledged or the context expires.
	Interrupt(ctx context.Context) error

	// Steerable returns whether this controller supports live steering.
	Steerable() bool
}

// noopController is the default controller for sessions that don't support
// live steering (sprites, adopted sessions, legacy).
type noopController struct{}

func (noopController) SendMessage(context.Context, string) error { return ErrNotSteerable }
func (noopController) Interrupt(context.Context) error           { return ErrNotSteerable }
func (noopController) Steerable() bool                           { return false }

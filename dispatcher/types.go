package dispatcher

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// DispatchRequest defines one session launch.
type DispatchRequest struct {
	Name                 string
	Prompt               string
	Provider             string
	Model                string
	Skill                string
	ReasoningLevel       string
	WorktreePath         string
	MaxTurns             int
	EnvVars              map[string]string
	BudgetCap            float64
	AllowPrimaryCheckout bool
	TaskKey              string // resolved task type key (e.g., "execute", "schedule")
	DomainSkill          string // for execute: adapter-configured domain skill
	Runtime              string // runtime kind from queue item (e.g., "tmux", "sprites")
	SystemPrompt         string // if set, used directly as system prompt — skips skill resolution
	DispatchWarning      string // set by factory on runtime fallback — carries the original dispatch error
}

// Validate ensures required request fields are set at the boundary.
func (r DispatchRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("session name not set")
	}
	if strings.TrimSpace(r.Prompt) == "" {
		return fmt.Errorf("prompt not set")
	}
	if strings.TrimSpace(r.Provider) == "" {
		return fmt.Errorf("provider not set")
	}
	if strings.TrimSpace(r.Model) == "" {
		return fmt.Errorf("model not set")
	}
	if strings.TrimSpace(r.WorktreePath) == "" {
		return fmt.Errorf("worktree path not set")
	}
	if r.MaxTurns < 0 {
		return fmt.Errorf("max turns is negative")
	}
	if r.BudgetCap < 0 {
		return fmt.Errorf("budget cap is negative")
	}
	return nil
}

// SessionEvent is a normalized live event emitted from a running session.
type SessionEvent struct {
	Type      string
	Message   string
	Timestamp time.Time
	Rationale string
	CostUSD   float64
	TokensIn  int
	TokensOut int
}

// Session is one dispatched agent session.
type Session interface {
	ID() string
	Status() string
	Events() <-chan SessionEvent
	Done() <-chan struct{}
	TotalCost() float64
	Kill() error
}

// Dispatcher starts sessions from requests.
type Dispatcher interface {
	Dispatch(ctx context.Context, req DispatchRequest) (Session, error)
}

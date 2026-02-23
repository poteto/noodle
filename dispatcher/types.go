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
	TaskKey              string // resolved task type key (e.g., "execute", "prioritize")
	DomainSkill          string // for execute: adapter-configured domain skill
	Runtime              string // command template from frontmatter, empty = built-in
}

// Validate ensures required request fields are set at the boundary.
func (r DispatchRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("session name is required")
	}
	if strings.TrimSpace(r.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	provider := strings.ToLower(strings.TrimSpace(r.Provider))
	switch provider {
	case "claude", "codex":
	default:
		return fmt.Errorf("provider must be claude or codex")
	}
	if strings.TrimSpace(r.Model) == "" {
		return fmt.Errorf("model is required")
	}
	if strings.TrimSpace(r.WorktreePath) == "" {
		return fmt.Errorf("worktree path is required")
	}
	if r.MaxTurns < 0 {
		return fmt.Errorf("max turns cannot be negative")
	}
	if r.BudgetCap < 0 {
		return fmt.Errorf("budget cap cannot be negative")
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

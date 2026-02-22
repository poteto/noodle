package spawner

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SpawnRequest defines one session launch.
type SpawnRequest struct {
	Name           string
	Prompt         string
	Provider       string
	Model          string
	Skill          string
	ReasoningLevel string
	WorktreePath   string
	MaxTurns       int
	EnvVars        map[string]string
	BudgetCap      float64
}

// Validate ensures required request fields are set at the boundary.
func (r SpawnRequest) Validate() error {
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

// Session is one spawned agent session.
type Session interface {
	ID() string
	Status() string
	Events() <-chan SessionEvent
	Done() <-chan struct{}
	TotalCost() float64
	Kill() error
}

// Spawner starts sessions from requests.
type Spawner interface {
	Spawn(ctx context.Context, req SpawnRequest) (Session, error)
}

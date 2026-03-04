package dispatcher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/poteto/noodle/event"
)

// ProviderConfig holds optional CLI binary directory and extra flags by provider.
type ProviderConfig struct {
	Path string
	Args []string
}

// ProviderConfigs holds per-provider configuration.
type ProviderConfigs struct {
	Claude ProviderConfig
	Codex  ProviderConfig
}

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
	Runtime              string // runtime kind from queue item (e.g., "process", "sprites")
	SystemPrompt         string // if set, used directly as system prompt — skips skill resolution
	DispatchWarning      string // set by factory on runtime fallback — carries the original dispatch error
	DisplayName          string // stable short name preserved across retries
	Title                string // queue item title for display
	RetryCount           int    // attempt number (0 = first try)
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

// SessionStatus classifies how a session ended.
type SessionStatus string

const (
	StatusCompleted SessionStatus = "completed"
	StatusFailed    SessionStatus = "failed"
	StatusCancelled SessionStatus = "cancelled"
	StatusKilled    SessionStatus = "killed"
)

// IsTerminal reports whether the status represents a finished session.
func (s SessionStatus) IsTerminal() bool {
	switch s {
	case StatusCompleted, StatusFailed, StatusCancelled, StatusKilled:
		return true
	default:
		return false
	}
}

// SessionOutcome captures terminal state classification and diagnostics.
type SessionOutcome struct {
	Status         SessionStatus
	Reason         string
	HasDeliverable bool
	ExitCode       int
}

func sessionStatusFromString(status string) SessionStatus {
	switch status {
	case string(StatusCompleted):
		return StatusCompleted
	case string(StatusFailed):
		return StatusFailed
	case string(StatusCancelled):
		return StatusCancelled
	case string(StatusKilled):
		return StatusKilled
	default:
		return ""
	}
}

// Session is one dispatched agent session.
type Session interface {
	ID() string
	Status() string
	Outcome() SessionOutcome
	Events() <-chan SessionEvent
	Done() <-chan struct{}
	TotalCost() float64
	Terminate() error
	ForceKill() error
	Controller() AgentController
}

// SessionEventSink receives real-time session events for broadcasting.
type SessionEventSink interface {
	PublishSessionEvent(sessionID string, ev event.Event)
	PublishSessionDelta(sessionID string, text string, ts time.Time)
}

// Dispatcher starts sessions from requests.
type Dispatcher interface {
	Dispatch(ctx context.Context, req DispatchRequest) (Session, error)
}

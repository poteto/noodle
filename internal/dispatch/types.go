package dispatch

import (
	"time"

	"github.com/poteto/noodle/internal/state"
)

// DispatchPlan is the planner output for a single scheduling cycle.
type DispatchPlan struct {
	Candidates        []DispatchCandidate `json:"candidates"`
	Blocked           []BlockedCandidate  `json:"blocked"`
	CapacityRemaining int                 `json:"capacity_remaining"`
}

// DispatchCandidate is a stage ready for dispatch.
type DispatchCandidate struct {
	OrderID    string          `json:"order_id"`
	StageIndex int             `json:"stage_index"`
	Stage      state.StageNode `json:"stage"`
	Runtime    string          `json:"runtime"`
}

// BlockedCandidate is a stage that could not be dispatched.
type BlockedCandidate struct {
	OrderID    string `json:"order_id"`
	StageIndex int    `json:"stage_index"`
	Reason     string `json:"reason"`
}

// CompletionRecord is the terminal outcome of one attempt.
type CompletionRecord struct {
	OrderID     string              `json:"order_id"`
	StageIndex  int                 `json:"stage_index"`
	AttemptID   string              `json:"attempt_id"`
	Status      state.AttemptStatus `json:"status"`
	ExitCode    *int                `json:"exit_code"`
	Error       string              `json:"error"`
	CompletedAt time.Time           `json:"completed_at"`
}

// RetryPolicy controls retry eligibility.
type RetryPolicy struct {
	MaxAttempts int
	ShouldRetry func(CompletionRecord) bool
}

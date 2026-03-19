package state

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/filex"
	"github.com/poteto/noodle/internal/statever"
)

// OrderLifecycleStatus is the lifecycle status of an order.
type OrderLifecycleStatus string

const (
	OrderPending   OrderLifecycleStatus = "pending"
	OrderActive    OrderLifecycleStatus = "active"
	OrderCompleted OrderLifecycleStatus = "completed"
	OrderFailed    OrderLifecycleStatus = "failed"
	OrderCancelled OrderLifecycleStatus = "cancelled"
)

// StageLifecycleStatus is the lifecycle status of a stage within an order.
type StageLifecycleStatus string

const (
	StagePending     StageLifecycleStatus = "pending"
	StageDispatching StageLifecycleStatus = "dispatching"
	StageRunning     StageLifecycleStatus = "running"
	StageMerging     StageLifecycleStatus = "merging"
	StageReview      StageLifecycleStatus = "review"
	StageCompleted   StageLifecycleStatus = "completed"
	StageFailed      StageLifecycleStatus = "failed"
	StageSkipped     StageLifecycleStatus = "skipped"
	StageCancelled   StageLifecycleStatus = "cancelled"
)

// AttemptStatus is the lifecycle status of a dispatch attempt.
type AttemptStatus string

const (
	AttemptLaunching AttemptStatus = "launching"
	AttemptRunning   AttemptStatus = "running"
	AttemptCompleted AttemptStatus = "completed"
	AttemptFailed    AttemptStatus = "failed"
	AttemptCancelled AttemptStatus = "cancelled"
)

// RunMode controls how the loop processes orders.
type RunMode string

const (
	RunModeAuto       RunMode = "auto"
	RunModeSupervised RunMode = "supervised"
	RunModeManual     RunMode = "manual"
)

// ModeEpoch is a monotonic counter incremented on every mode transition.
// Compatible with mode.ModeEpoch for use across package boundaries.
type ModeEpoch uint64

// ModeTransitionRecord tracks a single mode change for audit purposes.
// Compatible with mode.ModeTransitionRecord for use across package boundaries.
type ModeTransitionRecord struct {
	FromMode    RunMode   `json:"from_mode"`
	ToMode      RunMode   `json:"to_mode"`
	Epoch       ModeEpoch `json:"epoch"`
	RequestedBy string    `json:"requested_by"`
	Reason      string    `json:"reason"`
	AppliedAt   time.Time `json:"applied_at"`
}

// MaxModeTransitionHistory is the number of recent transitions retained.
const MaxModeTransitionHistory = 50

// State is the top-level canonical backend state. All loop scheduling
// decisions derive from this single model.
type State struct {
	Orders          map[string]OrderNode         `json:"orders"`
	PendingReviews  map[string]PendingReviewNode `json:"pending_reviews"`
	Mode            RunMode                      `json:"mode"`
	ModeEpoch       ModeEpoch                    `json:"mode_epoch"`
	ModeTransitions []ModeTransitionRecord       `json:"mode_transitions"`
	SchemaVersion   statever.SchemaVersion       `json:"schema_version"`
	LastEventID     string                       `json:"last_event_id"`
}

// OrderNode represents an order's canonical state.
type OrderNode struct {
	OrderID   string               `json:"order_id"`
	Status    OrderLifecycleStatus `json:"status"`
	Stages    []StageNode          `json:"stages"`
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
	Metadata  map[string]string    `json:"metadata"`
}

// StageNode represents a stage within an order.
type StageNode struct {
	StageIndex int                  `json:"stage_index"`
	Status     StageLifecycleStatus `json:"status"`
	Skill      string               `json:"skill"`
	Runtime    string               `json:"runtime"`
	Attempts   []AttemptNode        `json:"attempts"`
	Group      string               `json:"group"`
}

// AttemptNode represents a dispatch attempt for a stage.
type AttemptNode struct {
	AttemptID    string        `json:"attempt_id"`
	SessionID    string        `json:"session_id"`
	Status       AttemptStatus `json:"status"`
	StartedAt    time.Time     `json:"started_at"`
	CompletedAt  time.Time     `json:"completed_at"`
	ExitCode     *int          `json:"exit_code"`
	WorktreeName string        `json:"worktree_name"`
	Error        string        `json:"error"`
}

// PendingReviewNode records a stage parked for human review.
type PendingReviewNode struct {
	OrderID      string   `json:"order_id"`
	StageIndex   int      `json:"stage_index"`
	TaskKey      string   `json:"task_key"`
	Prompt       string   `json:"prompt"`
	Provider     string   `json:"provider"`
	Model        string   `json:"model"`
	Runtime      string   `json:"runtime"`
	Skill        string   `json:"skill"`
	Plan         []string `json:"plan"`
	WorktreeName string   `json:"worktree_name"`
	WorktreePath string   `json:"worktree_path"`
	SessionID    string   `json:"session_id"`
	Reason       string   `json:"reason"`
}

// IsTerminal reports whether the order status is terminal
// (no further transitions expected).
func (s OrderLifecycleStatus) IsTerminal() bool {
	switch s {
	case OrderCompleted, OrderFailed, OrderCancelled:
		return true
	}
	return false
}

// IsTerminal reports whether the stage status is terminal
// (no further transitions expected).
func (s StageLifecycleStatus) IsTerminal() bool {
	switch s {
	case StageCompleted, StageFailed, StageSkipped, StageCancelled:
		return true
	}
	return false
}

// IsBusy reports whether the stage is in an active/busy state.
// Busy and terminal are distinct categories — StagePending is NOT busy.
func (s StageLifecycleStatus) IsBusy() bool {
	switch s {
	case StageDispatching, StageRunning, StageMerging, StageReview:
		return true
	}
	return false
}

// Clone returns a deep copy of the State. Mutations to the clone do not
// affect the original.
func (s State) Clone() State {
	out := s

	// Deep-copy mode transitions slice.
	if s.ModeTransitions != nil {
		transitionsCopy := make([]ModeTransitionRecord, len(s.ModeTransitions))
		copy(transitionsCopy, s.ModeTransitions)
		out.ModeTransitions = transitionsCopy
	}

	if s.PendingReviews != nil {
		reviewsCopy := make(map[string]PendingReviewNode, len(s.PendingReviews))
		for orderID, review := range s.PendingReviews {
			reviewCopy := review
			if review.Plan != nil {
				planCopy := make([]string, len(review.Plan))
				copy(planCopy, review.Plan)
				reviewCopy.Plan = planCopy
			}
			reviewsCopy[orderID] = reviewCopy
		}
		out.PendingReviews = reviewsCopy
	}

	if s.Orders == nil {
		out.Orders = nil
		return out
	}

	out.Orders = make(map[string]OrderNode, len(s.Orders))
	for orderID, order := range s.Orders {
		orderCopy := order

		if order.Metadata != nil {
			metadataCopy := make(map[string]string, len(order.Metadata))
			for key, value := range order.Metadata {
				metadataCopy[key] = value
			}
			orderCopy.Metadata = metadataCopy
		}

		if order.Stages != nil {
			stagesCopy := make([]StageNode, len(order.Stages))
			for i := range order.Stages {
				stageCopy := order.Stages[i]
				if order.Stages[i].Attempts != nil {
					attemptsCopy := make([]AttemptNode, len(order.Stages[i].Attempts))
					for j := range order.Stages[i].Attempts {
						attemptCopy := order.Stages[i].Attempts[j]
						if attemptCopy.ExitCode != nil {
							exitCode := *attemptCopy.ExitCode
							attemptCopy.ExitCode = &exitCode
						}
						attemptsCopy[j] = attemptCopy
					}
					stageCopy.Attempts = attemptsCopy
				}
				stagesCopy[i] = stageCopy
			}
			orderCopy.Stages = stagesCopy
		}

		out.Orders[orderID] = orderCopy
	}

	return out
}

// LookupStage finds an order and stage by order ID and stage index.
// Returns the order, stage, and true if found; zero values and false otherwise.
func (s State) LookupStage(orderID string, stageIndex int) (OrderNode, StageNode, bool) {
	if stageIndex < 0 {
		return OrderNode{}, StageNode{}, false
	}
	order, ok := s.Orders[orderID]
	if !ok {
		return OrderNode{}, StageNode{}, false
	}
	if stageIndex >= len(order.Stages) {
		return OrderNode{}, StageNode{}, false
	}
	return order, order.Stages[stageIndex], true
}

// ClonedExitCode returns a deep copy of an exit code pointer.
func ClonedExitCode(code *int) *int {
	if code == nil {
		return nil
	}
	v := *code
	return &v
}

// OrderBusyIndex returns a map from order ID to stage index for stages in
// active states (dispatching, running, merging, review). If an order has
// multiple busy stages, only the first is recorded.
func (s *State) OrderBusyIndex() map[string]int {
	idx := make(map[string]int)
	for orderID, order := range s.Orders {
		for _, stage := range order.Stages {
			if stage.Status.IsBusy() {
				idx[orderID] = stage.StageIndex
				break
			}
		}
	}
	return idx
}

// AttemptBySessionIndex returns a map from session ID to AttemptNode pointer
// for all attempts in state. If multiple attempts share a session ID, the
// last one encountered wins.
func (s *State) AttemptBySessionIndex() map[string]*AttemptNode {
	idx := make(map[string]*AttemptNode)
	for orderID := range s.Orders {
		order := s.Orders[orderID]
		for i := range order.Stages {
			for j := range order.Stages[i].Attempts {
				a := &order.Stages[i].Attempts[j]
				if a.SessionID != "" {
					idx[a.SessionID] = a
				}
			}
		}
	}
	return idx
}

// PendingEffectIndex is a placeholder for Phase 3. Returns an empty map.
func (s *State) PendingEffectIndex() map[string]struct{} {
	return map[string]struct{}{}
}

// Persist writes the state to path using atomic file write.
func (s *State) Persist(path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode canonical state: %w", err)
	}
	if err := filex.WriteFileAtomic(path, append(data, '\n')); err != nil {
		return fmt.Errorf("persist canonical state: %w", err)
	}
	return nil
}

// Load reads canonical state from path. Returns a zero State and no error
// if the file does not exist (first run).
func Load(path string) (State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return State{}, fmt.Errorf("read canonical state: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return State{}, nil
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}, fmt.Errorf("corrupted canonical state at %s: %w", path, err)
	}
	return s, nil
}

// Validate checks lifecycle invariants of the state. Returns the first
// violation found, or nil if the state is consistent.
func (s *State) Validate() error {
	seenAttemptIDs := make(map[string]string) // attemptID -> "order/stage" for error messages

	for orderID, order := range s.Orders {
		// Active order must have at least one non-terminal stage.
		if order.Status == OrderActive {
			hasNonTerminal := false
			for _, stage := range order.Stages {
				if !stage.Status.IsTerminal() {
					hasNonTerminal = true
					break
				}
			}
			if !hasNonTerminal {
				return fmt.Errorf("order %q has status active but all stages are terminal", orderID)
			}
		}

		for i, stage := range order.Stages {
			// Stage indexes must be sequential starting at 0.
			if stage.StageIndex != i {
				return fmt.Errorf("order %q stage %d has index %d (expected %d)", orderID, i, stage.StageIndex, i)
			}

			// Running stage must have at least one running attempt.
			if stage.Status == StageRunning {
				hasRunningAttempt := false
				for _, a := range stage.Attempts {
					if a.Status == AttemptRunning {
						hasRunningAttempt = true
						break
					}
				}
				if !hasRunningAttempt {
					return fmt.Errorf("order %q stage %d has status running but no attempt is running", orderID, i)
				}
			}

			// AttemptIDs must be unique across the entire state.
			for _, a := range stage.Attempts {
				if a.AttemptID == "" {
					continue
				}
				loc := fmt.Sprintf("order %q stage %d", orderID, i)
				if prevLoc, exists := seenAttemptIDs[a.AttemptID]; exists {
					return fmt.Errorf("attempt %q appears in both %s and %s", a.AttemptID, prevLoc, loc)
				}
				seenAttemptIDs[a.AttemptID] = loc
			}
		}
	}

	return nil
}

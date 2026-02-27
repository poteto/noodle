package loop

import (
	"github.com/poteto/noodle/mise"
)

// CookSummary is a read-only summary of an active cook for the snapshot.
type CookSummary struct {
	SessionID   string `json:"session_id"`
	OrderID     string `json:"order_id"`
	TaskKey     string `json:"task_key"`
	Skill       string `json:"skill"`
	Runtime     string `json:"runtime"`
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	DisplayName string `json:"display_name"`
}

// LoopState is the immutable snapshot published at cycle boundaries.
// HTTP/SSE readers get lock-free access via atomic.Pointer.
type LoopState struct {
	Orders           []Order             `json:"orders"`
	ActiveCooks      []CookSummary       `json:"active_cooks"`
	PendingReviews   []PendingReviewItem `json:"pending_reviews"`
	ActiveSummary    mise.ActiveSummary  `json:"active_summary"`
	ActiveOrderIDs   []string            `json:"active_order_ids"`
	MaxCooks         int                 `json:"max_cooks"`
	Autonomy         string              `json:"autonomy"`
	LoopState        string              `json:"loop_state"`
	ActionNeeded     []string            `json:"action_needed,omitempty"`
}

// LoopStateProvider is the interface the server uses to read loop state.
type LoopStateProvider interface {
	State() LoopState
}

// State returns a read-only snapshot of current loop state.
func (l *Loop) State() LoopState {
	if ptr := l.stateSnapshot.Load(); ptr != nil {
		return *ptr
	}
	return l.buildState()
}

// publishState builds and stores an immutable snapshot at cycle boundaries.
func (l *Loop) publishState() {
	state := l.buildState()
	l.stateSnapshot.Store(&state)
}

func (l *Loop) buildState() LoopState {
	cooks := make([]CookSummary, 0, len(l.activeCooksByOrder))
	orderIDs := make([]string, 0, len(l.activeCooksByOrder))
	summary := mise.ActiveSummary{
		ByTaskKey: make(map[string]int),
		ByStatus:  make(map[string]int),
		ByRuntime: make(map[string]int),
	}
	for _, cook := range l.activeCooksByOrder {
		cooks = append(cooks, CookSummary{
			SessionID:   cook.session.ID(),
			OrderID:     cook.orderID,
			TaskKey:     cook.stage.TaskKey,
			Skill:       cook.stage.Skill,
			Runtime:     cook.stage.Runtime,
			Provider:    cook.stage.Provider,
			Model:       cook.stage.Model,
			DisplayName: cook.displayName,
		})
		orderIDs = append(orderIDs, cook.orderID)
		summary.Total++
		summary.ByTaskKey[cook.stage.TaskKey]++
		summary.ByRuntime[cook.stage.Runtime]++
	}

	orders := make([]Order, len(l.orders.Orders))
	copy(orders, l.orders.Orders)

	reviews := make([]PendingReviewItem, 0, len(l.pendingReview))
	for _, pr := range l.pendingReview {
		reviews = append(reviews, PendingReviewItem{
			OrderID:      pr.orderID,
			StageIndex:   pr.stageIndex,
			TaskKey:      pr.stage.TaskKey,
			WorktreeName: pr.worktreeName,
			WorktreePath: pr.worktreePath,
			SessionID:    pr.sessionID,
			Reason:       pr.reason,
		})
	}

	return LoopState{
		Orders:         orders,
		ActiveCooks:    cooks,
		PendingReviews: reviews,
		ActiveSummary:  summary,
		ActiveOrderIDs: orderIDs,
		MaxCooks:       l.config.Concurrency.MaxCooks,
		Autonomy:       l.config.Autonomy,
		LoopState:      string(l.state),
		ActionNeeded:   l.orders.ActionNeeded,
	}
}

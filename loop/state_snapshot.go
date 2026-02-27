package loop

import (
	"sort"
	"strings"
	"time"

	"github.com/poteto/noodle/mise"
)

type CookSummary struct {
	SessionID    string    `json:"session_id"`
	OrderID      string    `json:"order_id"`
	TaskKey      string    `json:"task_key,omitempty"`
	Skill        string    `json:"skill,omitempty"`
	Runtime      string    `json:"runtime,omitempty"`
	Provider     string    `json:"provider,omitempty"`
	Model        string    `json:"model,omitempty"`
	StartedAt    time.Time `json:"started_at"`
	DisplayName  string    `json:"display_name,omitempty"`
	Status       string    `json:"status"`
	TotalCostUSD float64   `json:"total_cost_usd"`
}

type LoopState struct {
	UpdatedAt          time.Time           `json:"updated_at"`
	Orders             []Order             `json:"orders"`
	ActiveCooks        []CookSummary       `json:"active_cooks"`
	PendingReviews     []PendingReviewItem `json:"pending_reviews"`
	PendingReviewCount int                 `json:"pending_review_count"`
	RecentHistory      []mise.HistoryItem  `json:"recent_history"`
	Status             string              `json:"status"`
	ActiveSummary      mise.ActiveSummary  `json:"active_summary"`
	ActiveOrderIDs     []string            `json:"active_order_ids"`
	TotalCostUSD       float64             `json:"total_cost_usd"`
	MaxCooks           int                 `json:"max_cooks"`
	Autonomy           string              `json:"autonomy"`
	ActionNeeded       []string            `json:"action_needed"`
}

func (l *Loop) publishState() {
	snapshot := l.buildLoopStateSnapshot()
	l.publishedState.Store(snapshot)
}

func (l *Loop) State() LoopState {
	current := l.publishedState.Load()
	if current == nil {
		return LoopState{}
	}
	return cloneLoopState(*current)
}

func (l *Loop) buildLoopStateSnapshot() *LoopState {
	ordersFile, _ := l.currentOrders()
	ordersCopy := make([]Order, 0, len(ordersFile.Orders))
	for _, order := range ordersFile.Orders {
		ordersCopy = append(ordersCopy, cloneOrder(order))
	}
	activeCooks := make([]CookSummary, 0, len(l.cooks.activeCooksByOrder))
	totalCost := 0.0
	for _, cook := range l.cooks.activeCooksByOrder {
		if cook == nil || cook.session == nil {
			continue
		}
		totalCost += cook.session.TotalCost()
		activeCooks = append(activeCooks, CookSummary{
			SessionID:    cook.session.ID(),
			OrderID:      cook.orderID,
			TaskKey:      cook.stage.TaskKey,
			Skill:        cook.stage.Skill,
			Runtime:      cook.stage.Runtime,
			Provider:     cook.stage.Provider,
			Model:        cook.stage.Model,
			StartedAt:    cook.startedAt,
			DisplayName:  cook.displayName,
			Status:       strings.ToLower(strings.TrimSpace(cook.session.Status())),
			TotalCostUSD: cook.session.TotalCost(),
		})
	}
	sort.Slice(activeCooks, func(i, j int) bool {
		return activeCooks[i].SessionID < activeCooks[j].SessionID
	})

	pendingReviews := make([]PendingReviewItem, 0, len(l.cooks.pendingReview))
	for _, pending := range l.cooks.pendingReview {
		if pending == nil {
			continue
		}
		pendingReviews = append(pendingReviews, PendingReviewItem{
			OrderID:      pending.orderID,
			StageIndex:   pending.stageIndex,
			TaskKey:      pending.stage.TaskKey,
			Prompt:       pending.stage.Prompt,
			Provider:     pending.stage.Provider,
			Model:        pending.stage.Model,
			Runtime:      pending.stage.Runtime,
			Skill:        pending.stage.Skill,
			Plan:         append([]string(nil), pending.plan...),
			WorktreeName: pending.worktreeName,
			WorktreePath: pending.worktreePath,
			SessionID:    pending.sessionID,
			Reason:       pending.reason,
		})
	}
	sort.Slice(pendingReviews, func(i, j int) bool {
		return pendingReviews[i].OrderID < pendingReviews[j].OrderID
	})

	return &LoopState{
		UpdatedAt:          l.deps.Now().UTC(),
		Orders:             ordersCopy,
		ActiveCooks:        activeCooks,
		PendingReviews:     pendingReviews,
		PendingReviewCount: len(pendingReviews),
		RecentHistory:      l.snapshotRecentHistory(),
		Status:             string(l.state),
		ActiveSummary:      l.snapshotActiveSummary(),
		ActiveOrderIDs:     ActiveOrderIDs(ordersFile),
		TotalCostUSD:       totalCost,
		MaxCooks:           l.config.Concurrency.MaxCooks,
		Autonomy:           l.config.Autonomy,
		ActionNeeded:       append([]string(nil), ordersFile.ActionNeeded...),
	}
}

func cloneLoopState(state LoopState) LoopState {
	cloned := state
	cloned.Orders = make([]Order, 0, len(state.Orders))
	for _, order := range state.Orders {
		cloned.Orders = append(cloned.Orders, cloneOrder(order))
	}
	cloned.ActiveCooks = append([]CookSummary(nil), state.ActiveCooks...)
	cloned.PendingReviews = make([]PendingReviewItem, 0, len(state.PendingReviews))
	for _, item := range state.PendingReviews {
		item.Plan = append([]string(nil), item.Plan...)
		cloned.PendingReviews = append(cloned.PendingReviews, item)
	}
	cloned.RecentHistory = append([]mise.HistoryItem(nil), state.RecentHistory...)
	cloned.ActiveOrderIDs = append([]string(nil), state.ActiveOrderIDs...)
	cloned.ActionNeeded = append([]string(nil), state.ActionNeeded...)
	cloned.ActiveSummary = mise.ActiveSummary{
		Total:     state.ActiveSummary.Total,
		ByTaskKey: make(map[string]int, len(state.ActiveSummary.ByTaskKey)),
		ByStatus:  make(map[string]int, len(state.ActiveSummary.ByStatus)),
		ByRuntime: make(map[string]int, len(state.ActiveSummary.ByRuntime)),
	}
	for k, v := range state.ActiveSummary.ByTaskKey {
		cloned.ActiveSummary.ByTaskKey[k] = v
	}
	for k, v := range state.ActiveSummary.ByStatus {
		cloned.ActiveSummary.ByStatus[k] = v
	}
	for k, v := range state.ActiveSummary.ByRuntime {
		cloned.ActiveSummary.ByRuntime[k] = v
	}
	return cloned
}

package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/poteto/noodle/internal/filex"
	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/orderx"
	"github.com/poteto/noodle/internal/state"
)

type PendingReviewItem struct {
	OrderID      string   `json:"order_id"`
	StageIndex   int      `json:"stage_index"`
	TaskKey      string   `json:"task_key,omitempty"`
	Title        string   `json:"title,omitempty"`
	Prompt       string   `json:"prompt,omitempty"`
	Provider     string   `json:"provider,omitempty"`
	Model        string   `json:"model,omitempty"`
	Runtime      string   `json:"runtime,omitempty"`
	Skill        string   `json:"skill,omitempty"`
	Plan         []string `json:"plan,omitempty"`
	Rationale    string   `json:"rationale,omitempty"`
	WorktreeName string   `json:"worktree_name"`
	WorktreePath string   `json:"worktree_path"`
	SessionID    string   `json:"session_id,omitempty"`
	Reason       string   `json:"reason,omitempty"`
}

type pendingReviewFile struct {
	Items []PendingReviewItem `json:"items"`
}

func pendingReviewFilePath(runtimeDir string) string {
	return filepath.Join(runtimeDir, "pending-review.json")
}

func ReadPendingReview(runtimeDir string) ([]PendingReviewItem, error) {
	path := pendingReviewFilePath(runtimeDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []PendingReviewItem{}, nil
		}
		return nil, err
	}

	var payload pendingReviewFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("pending review file unreadable at %s: %w", path, err)
	}
	if payload.Items == nil {
		return []PendingReviewItem{}, nil
	}
	return payload.Items, nil
}

func (l *Loop) parkPendingReview(cook *cookHandle, reason string) error {
	if err := l.ensureCanonicalOrderFromOrders(cook.orderID); err != nil {
		return err
	}
	if err := l.emitEventChecked(ingest.EventStageReviewParked, l.reviewPayloadForCook(cook, reason)); err != nil {
		return err
	}
	if err := l.mirrorLegacyOrderFromCanonical(cook.orderID); err != nil {
		return err
	}
	return l.syncPendingReviewProjection()
}

func (l *Loop) writePendingReview() error {
	items := make([]PendingReviewItem, 0, len(l.canonical.PendingReviews))
	for _, review := range l.canonical.PendingReviews {
		items = append(items, PendingReviewItem{
			OrderID:      review.OrderID,
			StageIndex:   review.StageIndex,
			TaskKey:      review.TaskKey,
			Title:        "",
			Prompt:       review.Prompt,
			Provider:     review.Provider,
			Model:        review.Model,
			Runtime:      review.Runtime,
			Skill:        review.Skill,
			Plan:         append([]string(nil), review.Plan...),
			WorktreeName: review.WorktreeName,
			WorktreePath: review.WorktreePath,
			SessionID:    review.SessionID,
			Reason:       review.Reason,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].OrderID < items[j].OrderID
	})
	payload := pendingReviewFile{Items: items}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return filex.WriteFileAtomic(pendingReviewFilePath(l.runtimeDir), append(data, '\n'))
}

func (l *Loop) loadPendingReview() error {
	next := make(map[string]*pendingReviewCook, len(l.canonical.PendingReviews))
	for _, review := range l.canonical.PendingReviews {
		id := strings.TrimSpace(review.OrderID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(review.WorktreeName)
		if name == "" {
			name = defaultPendingReviewWorktreeName(id, review.StageIndex, review.TaskKey)
		}
		next[id] = &pendingReviewCook{
			cookIdentity: cookIdentity{
				orderID:    id,
				stageIndex: review.StageIndex,
				stage: Stage{
					TaskKey:  strings.TrimSpace(review.TaskKey),
					Prompt:   strings.TrimSpace(review.Prompt),
					Skill:    strings.TrimSpace(review.Skill),
					Provider: strings.TrimSpace(review.Provider),
					Model:    strings.TrimSpace(review.Model),
					Runtime:  strings.TrimSpace(review.Runtime),
				},
				plan: append([]string(nil), review.Plan...),
			},
			worktreeName: name,
			worktreePath: strings.TrimSpace(review.WorktreePath),
			sessionID:    strings.TrimSpace(review.SessionID),
			reason:       strings.TrimSpace(review.Reason),
		}
	}
	l.cooks.pendingReview = next
	return nil
}

func defaultPendingReviewWorktreeName(orderID string, stageIndex int, taskKey string) string {
	name := cookBaseName(orderID, stageIndex, strings.TrimSpace(taskKey))
	if name == "" {
		return orderID
	}
	return name
}

// reconcilePendingReview removes pending review entries whose canonical order
// no longer exists.
func (l *Loop) reconcilePendingReview() error {
	if len(l.cooks.pendingReview) == 0 {
		return nil
	}
	pruned := false
	for id := range l.cooks.pendingReview {
		if _, ok := l.canonical.Orders[id]; !ok {
			l.logger.Warn("pruning stale pending review", "order", id)
			delete(l.cooks.pendingReview, id)
			delete(l.canonical.PendingReviews, id)
			pruned = true
		}
	}
	if pruned {
		if err := l.persistCanonicalCheckpoint(); err != nil {
			return err
		}
		return l.syncPendingReviewProjection()
	}
	return nil
}

func (l *Loop) syncPendingReviewProjection() error {
	if err := l.loadPendingReview(); err != nil {
		return err
	}
	return l.writePendingReview()
}

func (l *Loop) ensureCanonicalOrderFromOrders(orderID string) error {
	if err := l.ensureCanonicalLoadedFromBridge(); err != nil {
		return err
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("canonical sync requires order_id")
	}
	if _, ok := l.canonical.Orders[orderID]; ok {
		return nil
	}
	orders, err := l.currentOrders()
	if err != nil {
		return err
	}
	for _, order := range orders.Orders {
		if order.ID != orderID {
			continue
		}
		l.syncCanonicalOrderFromLegacy(order)
		return l.persistCanonicalCheckpoint()
	}
	return fmt.Errorf("canonical order %q missing from orders state", orderID)
}

func (l *Loop) reviewPayloadForCook(cook *cookHandle, reason string) map[string]any {
	sessionID := ""
	if cook.session != nil {
		sessionID = cook.session.ID()
	}
	attemptID := ""
	if cook.attempt > 0 || cook.session != nil {
		attemptID = dispatchAttemptID(cook.orderID, cook.stageIndex, cook.attempt)
	}
	return map[string]any{
		"order_id":      cook.orderID,
		"stage_index":   cook.stageIndex,
		"attempt_id":    attemptID,
		"session_id":    sessionID,
		"worktree_name": cook.worktreeName,
		"worktree_path": cook.worktreePath,
		"reason":        reason,
		"task_key":      cook.stage.TaskKey,
		"prompt":        cook.stage.Prompt,
		"provider":      cook.stage.Provider,
		"model":         cook.stage.Model,
		"runtime":       cook.stage.Runtime,
		"skill":         cook.stage.Skill,
		"plan":          append([]string(nil), cook.plan...),
	}
}

func canonicalStageStatusToLegacy(status state.StageLifecycleStatus) orderx.StageStatus {
	switch status {
	case state.StageDispatching, state.StageRunning, state.StageReview:
		return StageStatusActive
	case state.StageMerging:
		return StageStatusMerging
	case state.StageCompleted:
		return StageStatusCompleted
	case state.StageFailed:
		return StageStatusFailed
	case state.StageCancelled, state.StageSkipped:
		return StageStatusCancelled
	default:
		return StageStatusPending
	}
}

func canonicalOrderStatusToLegacy(status state.OrderLifecycleStatus) (orderx.OrderStatus, bool) {
	switch status {
	case state.OrderCompleted, state.OrderCancelled:
		return "", true
	case state.OrderFailed:
		return OrderStatusFailed, false
	default:
		return OrderStatusActive, false
	}
}

func (l *Loop) mirrorLegacyOrderFromCanonical(orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("mirror canonical order requires order_id")
	}
	node, exists := l.canonical.Orders[orderID]
	legacyStatus, remove := canonicalOrderStatusToLegacy(node.Status)
	if !exists {
		remove = true
	}
	return l.mutateProjectedMirrorState(func(orders *OrdersFile) (bool, error) {
		for i := range orders.Orders {
			if orders.Orders[i].ID != orderID {
				continue
			}
			if remove {
				orders.Orders = append(orders.Orders[:i], orders.Orders[i+1:]...)
				return true, nil
			}
			changed := false
			if orders.Orders[i].Status != legacyStatus {
				orders.Orders[i].Status = legacyStatus
				changed = true
			}
			limit := len(orders.Orders[i].Stages)
			if len(node.Stages) < limit {
				limit = len(node.Stages)
			}
			for si := 0; si < limit; si++ {
				legacyStage := canonicalStageStatusToLegacy(node.Stages[si].Status)
				if orders.Orders[i].Stages[si].Status != legacyStage {
					orders.Orders[i].Stages[si].Status = legacyStage
					changed = true
				}
			}
			return changed, nil
		}
		if remove {
			return false, nil
		}
		return false, fmt.Errorf("mirror canonical order %q missing from orders state", orderID)
	})
}

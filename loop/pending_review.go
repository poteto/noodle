package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/poteto/noodle/internal/filex"
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
		// Old-format file — attempt to extract worktree path for manual resolution.
		var raw map[string]json.RawMessage
		if jsonErr := json.Unmarshal(data, &raw); jsonErr == nil {
			if itemsRaw, ok := raw["items"]; ok {
				var items []json.RawMessage
				if arrErr := json.Unmarshal(itemsRaw, &items); arrErr == nil {
					for _, itemRaw := range items {
						var partial struct {
							WorktreePath string `json:"worktree_path"`
						}
						if pErr := json.Unmarshal(itemRaw, &partial); pErr == nil && partial.WorktreePath != "" {
							logWarnf("pending review file has old format — worktree at %s needs manual merge or cleanup", partial.WorktreePath)
						}
					}
				}
			}
		}
		return []PendingReviewItem{}, nil
	}
	if payload.Items == nil {
		return []PendingReviewItem{}, nil
	}
	return payload.Items, nil
}

func (l *Loop) parkPendingReview(cook *cookHandle, reason string) error {
	l.pendingReview[cook.orderID] = &pendingReviewCook{
		orderID:      cook.orderID,
		stageIndex:   cook.stageIndex,
		stage:        cook.stage,
		plan:         cook.plan,
		worktreeName: cook.worktreeName,
		worktreePath: cook.worktreePath,
		sessionID:    cook.session.ID(),
		reason:       reason,
	}
	return l.writePendingReview()
}

func (l *Loop) writePendingReview() error {
	items := make([]PendingReviewItem, 0, len(l.pendingReview))
	for _, pending := range l.pendingReview {
		if pending == nil {
			l.logger.Warn("nil entry in pendingReview map")
			continue
		}
		items = append(items, PendingReviewItem{
			OrderID:      pending.orderID,
			StageIndex:   pending.stageIndex,
			TaskKey:      pending.stage.TaskKey,
			Title:        "",
			Prompt:       pending.stage.Prompt,
			Provider:     pending.stage.Provider,
			Model:        pending.stage.Model,
			Runtime:      pending.stage.Runtime,
			Skill:        pending.stage.Skill,
			Plan:         pending.plan,
			WorktreeName: pending.worktreeName,
			WorktreePath: pending.worktreePath,
			SessionID:    pending.sessionID,
			Reason:       pending.reason,
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
	items, err := ReadPendingReview(l.runtimeDir)
	if err != nil {
		return err
	}

	next := make(map[string]*pendingReviewCook, len(items))
	for _, item := range items {
		id := strings.TrimSpace(item.OrderID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(item.WorktreeName)
		if name == "" {
			continue
		}
		next[id] = &pendingReviewCook{
			orderID:    id,
			stageIndex: item.StageIndex,
			stage: Stage{
				TaskKey:  strings.TrimSpace(item.TaskKey),
				Prompt:   strings.TrimSpace(item.Prompt),
				Skill:    strings.TrimSpace(item.Skill),
				Provider: strings.TrimSpace(item.Provider),
				Model:    strings.TrimSpace(item.Model),
				Runtime:  strings.TrimSpace(item.Runtime),
			},
			plan:         item.Plan,
			worktreeName: name,
			worktreePath: strings.TrimSpace(item.WorktreePath),
			sessionID:    strings.TrimSpace(item.SessionID),
			reason:       strings.TrimSpace(item.Reason),
		}
	}
	l.pendingReview = next
	return nil
}

// reconcilePendingReview removes pending review entries whose order no longer
// exists in orders.json. This covers the crash window between advancing
// orders.json and updating pending-review.json.
func (l *Loop) reconcilePendingReview() error {
	if len(l.pendingReview) == 0 {
		return nil
	}
	orders, err := readOrders(l.deps.OrdersFile)
	if err != nil {
		return nil // no orders file yet — nothing to reconcile
	}
	orderIDs := make(map[string]struct{}, len(orders.Orders))
	for _, o := range orders.Orders {
		orderIDs[o.ID] = struct{}{}
	}
	pruned := false
	for id := range l.pendingReview {
		if _, ok := orderIDs[id]; !ok {
			l.logger.Warn("pruning stale pending review", "order", id)
			delete(l.pendingReview, id)
			pruned = true
		}
	}
	if pruned {
		return l.writePendingReview()
	}
	return nil
}

// logWarnf logs a warning to stderr. Used for degraded parse situations.
func logWarnf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "WARNING: "+format+"\n", args...)
}

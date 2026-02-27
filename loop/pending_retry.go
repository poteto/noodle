package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/poteto/noodle/internal/filex"
	"github.com/poteto/noodle/internal/orderx"
)

// PendingRetryItem is the serializable form of pendingRetryCook.
type PendingRetryItem struct {
	OrderID     string             `json:"order_id"`
	StageIndex  int                `json:"stage_index"`
	IsOnFailure bool               `json:"is_on_failure,omitempty"`
	OrderStatus orderx.OrderStatus `json:"order_status"`
	TaskKey     string `json:"task_key,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
	Provider    string `json:"provider,omitempty"`
	Model       string `json:"model,omitempty"`
	Runtime     string `json:"runtime,omitempty"`
	Skill       string `json:"skill,omitempty"`
	Plan        []string `json:"plan,omitempty"`
	Attempt     int    `json:"attempt"`
	DisplayName string `json:"display_name,omitempty"`
}

type pendingRetryFile struct {
	Items []PendingRetryItem `json:"items"`
}

func pendingRetryFilePath(runtimeDir string) string {
	return filepath.Join(runtimeDir, "pending-retry.json")
}

func (l *Loop) writePendingRetry() error {
	items := make([]PendingRetryItem, 0, len(l.pendingRetry))
	for _, p := range l.pendingRetry {
		if p == nil {
			continue
		}
		items = append(items, PendingRetryItem{
			OrderID:     p.orderID,
			StageIndex:  p.stageIndex,
			IsOnFailure: p.isOnFailure,
			OrderStatus: p.orderStatus,
			TaskKey:     p.stage.TaskKey,
			Prompt:      p.stage.Prompt,
			Provider:    p.stage.Provider,
			Model:       p.stage.Model,
			Runtime:     p.stage.Runtime,
			Skill:       p.stage.Skill,
			Plan:        p.plan,
			Attempt:     p.attempt,
			DisplayName: p.displayName,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].OrderID < items[j].OrderID
	})
	payload := pendingRetryFile{Items: items}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return filex.WriteFileAtomic(pendingRetryFilePath(l.runtimeDir), append(data, '\n'))
}

func readPendingRetryFile(runtimeDir string) ([]PendingRetryItem, error) {
	path := pendingRetryFilePath(runtimeDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var payload pendingRetryFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, nil // corrupt file — return empty
	}
	return payload.Items, nil
}

func (l *Loop) loadPendingRetry() error {
	path := pendingRetryFilePath(l.runtimeDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var payload pendingRetryFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil // corrupt file — start fresh
	}
	next := make(map[string]*pendingRetryCook, len(payload.Items))
	for _, item := range payload.Items {
		id := strings.TrimSpace(item.OrderID)
		if id == "" {
			continue
		}
		next[id] = &pendingRetryCook{
			cookIdentity: cookIdentity{
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
				plan: item.Plan,
			},
			isOnFailure: item.IsOnFailure,
			orderStatus: item.OrderStatus,
			attempt:     item.Attempt,
			displayName: item.DisplayName,
		}
	}
	l.pendingRetry = next
	return nil
}

// reconcilePendingRetry removes pending retry entries whose order no longer
// exists in orders.json or that have a live session (already recovered).
func (l *Loop) reconcilePendingRetry() error {
	if len(l.pendingRetry) == 0 {
		return nil
	}
	orders, err := l.currentOrders()
	if err != nil {
		return nil
	}
	orderIDs := make(map[string]struct{}, len(orders.Orders))
	for _, o := range orders.Orders {
		orderIDs[o.ID] = struct{}{}
	}
	// Remove retries for orders that no longer exist or are already active
	// (recovered session is handling them).
	pruned := false
	for id := range l.pendingRetry {
		if _, ok := orderIDs[id]; !ok {
			l.logger.Warn("pruning stale pending retry", "order", id)
			delete(l.pendingRetry, id)
			pruned = true
			continue
		}
		// If an adopted session is already handling this order, don't retry.
		if _, adopted := l.adoptedTargets[id]; adopted {
			l.logger.Info("pending retry superseded by recovered session", "order", id)
			delete(l.pendingRetry, id)
			pruned = true
		}
	}
	if pruned {
		return l.writePendingRetry()
	}
	return nil
}

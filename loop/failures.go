package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/poteto/noodle/internal/filex"
)

func (l *Loop) failedPath() string {
	return filepath.Join(l.runtimeDir, "failed.json")
}

func (l *Loop) loadFailedTargets() error {
	path := l.failedPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read failed targets: %w", err)
	}
	var failed map[string]string
	if err := json.Unmarshal(data, &failed); err != nil {
		return fmt.Errorf("parse failed targets: %w", err)
	}
	if l.cooks.failedTargets == nil {
		l.cooks.failedTargets = map[string]string{}
	}
	for id, reason := range failed {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		// Schedule is a system order, not a sticky user failure target.
		// Ignore stale entries so scheduler dispatch is never blocked on restart.
		if strings.EqualFold(id, scheduleOrderID) {
			continue
		}
		l.cooks.failedTargets[id] = strings.TrimSpace(reason)
	}
	return nil
}

func (l *Loop) markFailed(id string, reason string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	// Schedule failures should not become sticky failed targets.
	if strings.EqualFold(id, scheduleOrderID) {
		return nil
	}
	if l.cooks.failedTargets == nil {
		l.cooks.failedTargets = map[string]string{}
	}
	l.cooks.failedTargets[id] = strings.TrimSpace(reason)
	return l.writeFailedTargets()
}

func (l *Loop) writeFailedTargets() error {
	return l.writeFailedTargetsMap(l.cooks.failedTargets)
}

func (l *Loop) writeFailedTargetsMap(failed map[string]string) error {
	path := l.failedPath()
	if failed == nil {
		failed = map[string]string{}
	}
	data, err := json.MarshalIndent(failed, "", "  ")
	if err != nil {
		return fmt.Errorf("encode failed targets: %w", err)
	}
	if err := filex.WriteFileAtomic(path, append(data, '\n')); err != nil {
		return fmt.Errorf("rename failed targets file: %w", err)
	}
	return nil
}

// clearFailedTargetsForQueuedOrders removes sticky failed-target entries when
// the same order ID is queued in orders.json again.
func (l *Loop) clearFailedTargetsForQueuedOrders(orders OrdersFile) error {
	if len(l.cooks.failedTargets) == 0 {
		return nil
	}

	queued := make(map[string]struct{}, len(orders.Orders))
	for _, order := range orders.Orders {
		id := strings.TrimSpace(order.ID)
		if id == "" || strings.EqualFold(id, scheduleOrderID) {
			continue
		}
		queued[id] = struct{}{}
	}
	if len(queued) == 0 {
		return nil
	}

	nextFailed := make(map[string]string, len(l.cooks.failedTargets))
	cleared := make([]string, 0)
	for id, reason := range l.cooks.failedTargets {
		if _, ok := queued[id]; ok {
			cleared = append(cleared, id)
			continue
		}
		nextFailed[id] = reason
	}
	if len(cleared) == 0 {
		return nil
	}

	// Persist before mutating in-memory state to avoid divergence on write errors.
	if err := l.writeFailedTargetsMap(nextFailed); err != nil {
		return err
	}
	l.cooks.failedTargets = nextFailed

	slices.Sort(cleared)
	for _, orderID := range cleared {
		_ = l.events.Emit(LoopEventOrderRequeued, OrderRequeuedPayload{
			OrderID: orderID,
		})
	}
	return nil
}

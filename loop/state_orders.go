package loop

import (
	"fmt"
	"path/filepath"

	"github.com/poteto/noodle/internal/mode"
	"github.com/poteto/noodle/internal/orderx"
	"github.com/poteto/noodle/internal/projection"
	"github.com/poteto/noodle/internal/reducer"
	"github.com/poteto/noodle/internal/state"
)

func (l *Loop) loadOrdersState() error {
	orders, err := readOrders(l.deps.OrdersFile)
	if err != nil {
		return err
	}
	l.orders = cloneOrdersFile(orders)
	l.ordersLoaded = true
	return nil
}

func (l *Loop) currentOrders() (OrdersFile, error) {
	if !l.ordersLoaded {
		if err := l.loadOrdersState(); err != nil {
			return OrdersFile{}, err
		}
	}
	return cloneOrdersFile(l.orders), nil
}

func (l *Loop) writeOrdersState(orders OrdersFile) error {
	l.orders = cloneOrdersFile(orders)
	l.ordersLoaded = true
	if err := l.ensureCanonicalLoadedFromBridge(); err != nil {
		return err
	}
	if l.canonicalLoaded {
		l.syncCanonicalStateFromOrders(l.orders)
		if err := l.persistCanonicalCheckpoint(); err != nil {
			return err
		}
	}
	return l.writeProjectionState()
}

func (l *Loop) writeProjectedMirrorState(orders OrdersFile) error {
	l.orders = cloneOrdersFile(orders)
	l.ordersLoaded = true
	return l.writeProjectionState()
}

func (l *Loop) ensureCanonicalLoadedFromBridge() error {
	if l.canonicalLoaded {
		return nil
	}
	pendingReview := make([]PendingReviewItem, 0, len(l.cooks.pendingReview))
	for _, pending := range l.cooks.pendingReview {
		if pending == nil {
			continue
		}
		pendingReview = append(pendingReview, PendingReviewItem{
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
	now := timeNowUTC(l.deps.Now)
	l.canonical = synthesizeCanonicalState(l.orders, pendingReview, state.RunMode(l.config.Mode), now)
	l.effectLedger = reducer.NewEffectLedger()
	l.eventCounter.Store(0)
	l.canonicalLoaded = true
	return l.persistCanonicalCheckpoint()
}

func (l *Loop) mutateOrdersState(mutator func(*OrdersFile) (bool, error)) error {
	orders, err := l.currentOrders()
	if err != nil {
		return err
	}
	changed, err := mutator(&orders)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	if err := l.writeOrdersState(orders); err != nil {
		return err
	}
	return nil
}

func (l *Loop) mutateProjectedMirrorState(mutator func(*OrdersFile) (bool, error)) error {
	orders, err := l.currentOrders()
	if err != nil {
		return err
	}
	changed, err := mutator(&orders)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	return l.writeProjectedMirrorState(orders)
}

func (l *Loop) setOrdersState(orders OrdersFile) {
	l.orders = cloneOrdersFile(orders)
	l.ordersLoaded = true
}

func (l *Loop) ensureOrderStageStatus(orderID string, stageIndex int, status orderx.StageStatus) error {
	if orderID == "" {
		return fmt.Errorf("order id not set")
	}
	return l.mutateProjectedMirrorState(func(orders *OrdersFile) (bool, error) {
		for i := range orders.Orders {
			if orders.Orders[i].ID != orderID {
				continue
			}
			if stageIndex >= 0 && stageIndex < len(orders.Orders[i].Stages) {
				orders.Orders[i].Stages[stageIndex].Status = status
			}
			return true, nil
		}
		return true, nil
	})
}

// flushState writes all externally visible runtime artifacts atomically in a
// fixed order.
func (l *Loop) flushState() error {
	if err := l.writeProjectionState(); err != nil {
		return fmt.Errorf("flush projection: %w", err)
	}
	if l.TestFlushBarrier != nil {
		l.TestFlushBarrier()
	}
	if err := l.writePendingReview(); err != nil {
		return fmt.Errorf("flush pending review: %w", err)
	}
	if l.TestFlushBarrier != nil {
		l.TestFlushBarrier()
	}
	if err := l.writeLastAppliedSeq(); err != nil {
		return fmt.Errorf("flush last-applied-seq: %w", err)
	}
	return nil
}

func (l *Loop) writeProjectionState() error {
	if err := l.ensureCanonicalLoadedFromBridge(); err != nil {
		return err
	}
	if !l.canonicalLoaded {
		return nil
	}
	bundle, err := projection.Project(l.canonical, mode.ModeState{
		EffectiveMode: l.canonical.Mode,
		Epoch:         l.canonical.ModeEpoch,
	})
	if err != nil {
		return err
	}
	return projection.WriteProjectionFiles(l.stateOutputDir(), bundle)
}

func (l *Loop) stateOutputDir() string {
	if l.runtimeDir != "" {
		return l.runtimeDir
	}
	if l.deps.OrdersFile != "" {
		return filepath.Dir(l.deps.OrdersFile)
	}
	return ""
}

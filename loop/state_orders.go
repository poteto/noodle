package loop

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/orderx"
	"github.com/poteto/noodle/internal/statever"
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
	if err := writeOrdersAtomic(l.deps.OrdersFile, orders); err != nil {
		return err
	}
	l.orders = cloneOrdersFile(orders)
	l.ordersLoaded = true
	return nil
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

func (l *Loop) setOrdersState(orders OrdersFile) {
	l.orders = cloneOrdersFile(orders)
	l.ordersLoaded = true
}

func (l *Loop) ensureOrderStageStatus(orderID string, stageIndex int, status orderx.StageStatus) error {
	if orderID == "" {
		return fmt.Errorf("order id not set")
	}
	return l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
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

// flushState writes all in-memory state files atomically in a fixed order.
// Orders file is written first (source of truth). Each file uses
// write-to-temp + rename for atomic replacement.
func (l *Loop) flushState() error {
	if l.ordersLoaded {
		if err := writeOrdersAtomic(l.deps.OrdersFile, l.orders); err != nil {
			return fmt.Errorf("flush orders: %w", err)
		}
	}
	if err := l.writeStateMarker(); err != nil {
		return fmt.Errorf("flush state marker: %w", err)
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

func (l *Loop) writeStateMarker() error {
	path := l.stateMarkerPath()
	if path == "" {
		return nil
	}
	now := time.Now
	if l.deps.Now != nil {
		now = l.deps.Now
	}
	return statever.Write(path, statever.StateMarker{
		SchemaVersion: statever.Current,
		GeneratedAt:   now().UTC(),
	})
}

func (l *Loop) stateMarkerPath() string {
	if runtimeDir := strings.TrimSpace(l.runtimeDir); runtimeDir != "" {
		return filepath.Join(runtimeDir, "state.json")
	}
	if ordersPath := strings.TrimSpace(l.deps.OrdersFile); ordersPath != "" {
		return filepath.Join(filepath.Dir(ordersPath), "state.json")
	}
	return ""
}

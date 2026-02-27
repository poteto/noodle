package loop

import (
	"fmt"

	"github.com/poteto/noodle/internal/orderx"
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

func (l *Loop) mutateOrdersState(mutator func(*OrdersFile) error) error {
	orders, err := l.currentOrders()
	if err != nil {
		return err
	}
	if err := mutator(&orders); err != nil {
		return err
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

func (l *Loop) ensureOrderStageStatus(orderID string, stageIndex int, isOnFailure bool, status orderx.StageStatus) error {
	if orderID == "" {
		return fmt.Errorf("order id not set")
	}
	return l.mutateOrdersState(func(orders *OrdersFile) error {
		for i := range orders.Orders {
			if orders.Orders[i].ID != orderID {
				continue
			}
			stages := &orders.Orders[i].Stages
			if isOnFailure {
				stages = &orders.Orders[i].OnFailure
			}
			if stageIndex >= 0 && stageIndex < len(*stages) {
				(*stages)[stageIndex].Status = status
			}
			return nil
		}
		return nil
	})
}

// flushState writes all in-memory state files atomically in a fixed order.
// Orders file is written first (source of truth). Failed targets are
// independently durable (not derivable from orders). Each file uses
// write-to-temp + rename for atomic replacement.
func (l *Loop) flushState() error {
	if l.ordersLoaded {
		if err := writeOrdersAtomic(l.deps.OrdersFile, l.orders); err != nil {
			return fmt.Errorf("flush orders: %w", err)
		}
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
	if err := l.writeFailedTargets(); err != nil {
		return fmt.Errorf("flush failed targets: %w", err)
	}
	if l.TestFlushBarrier != nil {
		l.TestFlushBarrier()
	}
	if err := l.writePendingRetry(); err != nil {
		return fmt.Errorf("flush pending retry: %w", err)
	}
	if l.TestFlushBarrier != nil {
		l.TestFlushBarrier()
	}
	if err := l.writeLastAppliedSeq(); err != nil {
		return fmt.Errorf("flush last-applied-seq: %w", err)
	}
	return nil
}

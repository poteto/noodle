package loop

import "fmt"

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

func (l *Loop) ensureOrderStageStatus(orderID string, stageIndex int, isOnFailure bool, status string) error {
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

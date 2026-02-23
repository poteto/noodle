package loop

import (
	"strings"

	"github.com/poteto/noodle/mise"
)

type spawnPlanInput struct {
	QueueItems      []QueueItem
	Capacity        int
	ActiveCount     int
	AdoptedCount    int
	BlockingActive  bool
	BusyTargets     map[string]struct{}
	FailedTargets   map[string]struct{}
	AdoptedTargets  map[string]struct{}
	TicketedTargets map[string]struct{}
	IsBlocking      func(QueueItem) bool
}

func activeTicketTargetSet(brief mise.Brief) map[string]struct{} {
	targets := make(map[string]struct{}, len(brief.Tickets))
	for _, ticket := range brief.Tickets {
		target := strings.TrimSpace(ticket.Target)
		if target == "" {
			continue
		}
		switch ticket.Status {
		case "active", "blocked":
			targets[target] = struct{}{}
		}
	}
	return targets
}

func planSpawnItems(input spawnPlanInput) []QueueItem {
	limit := input.Capacity
	if limit <= 0 {
		limit = 1
	}

	current := input.ActiveCount + input.AdoptedCount
	blockingActive := input.BlockingActive
	plan := make([]QueueItem, 0, len(input.QueueItems))
	for _, item := range input.QueueItems {
		if blockingActive {
			break
		}
		if current >= limit {
			break
		}
		if input.IsBlocking != nil && input.IsBlocking(item) && current > 0 {
			continue
		}
		if _, busy := input.BusyTargets[item.ID]; busy {
			continue
		}
		if _, failed := input.FailedTargets[item.ID]; failed {
			continue
		}
		if _, adopted := input.AdoptedTargets[item.ID]; adopted {
			continue
		}
		if _, ticketed := input.TicketedTargets[item.ID]; ticketed {
			continue
		}

		plan = append(plan, item)
		current++
		if input.IsBlocking != nil && input.IsBlocking(item) {
			blockingActive = true
		}
	}
	return plan
}

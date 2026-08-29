package loop

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/internal/orderx"
	"github.com/poteto/noodle/mise"
)

// scheduleDecisionState is the canonical projection of everything a scheduling
// decision depends on. It is JSON-encoded and hashed to produce the digest that
// memoizes a schedule decision.
type scheduleDecisionState struct {
	Backlog   []backlogDigestItem    `json:"backlog"`
	Tickets   []ticketDigestItem     `json:"tickets"`
	Orders    []orderDigestItem      `json:"orders"`
	TaskTypes []mise.TaskTypeSummary `json:"task_types"`
	Routing   mise.RoutingSnapshot   `json:"routing"`
	Warnings  []string               `json:"warnings"`
}

type backlogDigestItem struct {
	ID     string         `json:"id"`
	Title  string         `json:"title"`
	Status string         `json:"status"`
	Plan   string         `json:"plan"`
	Extra  map[string]any `json:"extra,omitempty"`
}

type ticketDigestItem struct {
	Target     string `json:"target"`
	TargetType string `json:"target_type"`
	Status     string `json:"status"`
	BlockedBy  string `json:"blocked_by"`
	Reason     string `json:"reason"`
}

type orderDigestItem struct {
	ID     string   `json:"id"`
	Status string   `json:"status"`
	Stages []string `json:"stages"`
}

// scheduleDecisionDigest fingerprints the state a scheduling decision depends
// on: backlog identity, state and adapter extras (where dependencies live),
// tickets blocking targets, active non-schedule orders, schedulable task types,
// routing, and sync warnings.
//
// Everything the schedule cycle itself mutates is excluded — generated
// timestamps, active summary, resource counts, recent history and recent events
// — so dispatching, completing, or emitting `schedule.completed` for a schedule
// order cannot invalidate the scheduler's own previous decision. Stage statuses
// collapse for the same reason — see stageDigestState.
func scheduleDecisionDigest(brief mise.Brief, orders OrdersFile) (string, error) {
	state := scheduleDecisionState{
		Backlog:   backlogDigestItems(brief.Backlog),
		Tickets:   ticketDigestItems(brief.Tickets),
		Orders:    orderDigestItems(orders),
		TaskTypes: taskTypeDigestItems(brief.TaskTypes),
		Routing:   brief.Routing,
		Warnings:  append([]string{}, brief.Warnings...),
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("encode schedule decision state: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func backlogDigestItems(items []adapter.BacklogItem) []backlogDigestItem {
	digest := make([]backlogDigestItem, 0, len(items))
	for _, item := range items {
		digest = append(digest, backlogDigestItem{
			ID:     item.ID,
			Title:  item.Title,
			Status: item.Status,
			Plan:   item.Plan,
			Extra:  item.Extra,
		})
	}
	sort.Slice(digest, func(i, j int) bool { return digest[i].ID < digest[j].ID })
	return digest
}

func ticketDigestItems(tickets []event.Ticket) []ticketDigestItem {
	digest := make([]ticketDigestItem, 0, len(tickets))
	for _, ticket := range tickets {
		digest = append(digest, ticketDigestItem{
			Target:     ticket.Target,
			TargetType: string(ticket.TargetType),
			Status:     string(ticket.Status),
			BlockedBy:  ticket.BlockedBy,
			Reason:     ticket.Reason,
		})
	}
	sort.Slice(digest, func(i, j int) bool {
		if digest[i].Target != digest[j].Target {
			return digest[i].Target < digest[j].Target
		}
		return digest[i].Status < digest[j].Status
	})
	return digest
}

func orderDigestItems(orders OrdersFile) []orderDigestItem {
	digest := make([]orderDigestItem, 0, len(orders.Orders))
	for _, order := range orders.Orders {
		if isScheduleOrder(order) || isScheduleBootstrapOrder(order) {
			continue
		}
		stages := make([]string, 0, len(order.Stages))
		for _, stage := range order.Stages {
			stages = append(stages, stage.TaskKey+":"+stageDigestState(stage.Status))
		}
		digest = append(digest, orderDigestItem{
			ID:     order.ID,
			Status: string(order.Status),
			Stages: stages,
		})
	}
	sort.Slice(digest, func(i, j int) bool { return digest[i].ID < digest[j].ID })
	return digest
}

// stageDigestState collapses stage statuses to what a scheduling decision
// turns on. A stage moving pending → active → merging → completed is the loop
// executing the plan the scheduler already made; only a failed or cancelled
// stage is news the scheduler has to react to. Order completion is covered by
// the order's own status.
func stageDigestState(status orderx.StageStatus) string {
	switch status {
	case StageStatusFailed, StageStatusCancelled:
		return string(status)
	default:
		return "open"
	}
}

func taskTypeDigestItems(taskTypes []mise.TaskTypeSummary) []mise.TaskTypeSummary {
	digest := append([]mise.TaskTypeSummary{}, taskTypes...)
	sort.Slice(digest, func(i, j int) bool { return digest[i].Key < digest[j].Key })
	return digest
}

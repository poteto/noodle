package loop

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/internal/filex"
	"github.com/poteto/noodle/mise"
)

const scheduleEmptyMemoVersion = 1

type scheduleEmptyMemo struct {
	Version int    `json:"version"`
	Digest  string `json:"decision_digest"`
}

type scheduleDecisionTicket struct {
	Target     string   `json:"target"`
	TargetType string   `json:"target_type"`
	CookID     string   `json:"cook_id"`
	Files      []string `json:"files,omitempty"`
	Status     string   `json:"status"`
	BlockedBy  string   `json:"blocked_by,omitempty"`
	Reason     string   `json:"reason,omitempty"`
}

type scheduleDecisionEvent struct {
	Seq     uint64 `json:"seq"`
	Type    string `json:"type"`
	Summary string `json:"summary"`
}

type scheduleDecisionState struct {
	Backlog        []adapter.BacklogItem    `json:"backlog"`
	ActiveOrders   []Order                  `json:"active_non_schedule_orders"`
	ActionNeeded   []string                 `json:"action_needed,omitempty"`
	Tickets        []scheduleDecisionTicket `json:"tickets"`
	MaxConcurrency int                      `json:"max_concurrency"`
	Routing        mise.RoutingSnapshot     `json:"routing"`
	TaskTypes      []mise.TaskTypeSummary   `json:"task_types"`
	Warnings       []string                 `json:"warnings,omitempty"`
	LatestEvent    *scheduleDecisionEvent   `json:"latest_relevant_event,omitempty"`
}

func (l *Loop) scheduleEmptyMemoPath() string {
	return filepath.Join(l.runtimeDir, "schedule-empty-decision.json")
}

func (l *Loop) scheduleDecisionDigest(brief mise.Brief, orders OrdersFile) (string, error) {
	state := scheduleDecisionState{
		Backlog:        append([]adapter.BacklogItem(nil), brief.Backlog...),
		ActionNeeded:   append([]string(nil), orders.ActionNeeded...),
		MaxConcurrency: brief.Resources.MaxConcurrency,
		Routing: mise.RoutingSnapshot{
			Defaults:          brief.Routing.Defaults,
			AvailableRuntimes: append([]string(nil), brief.Routing.AvailableRuntimes...),
		},
		TaskTypes: append([]mise.TaskTypeSummary(nil), brief.TaskTypes...),
		Warnings:  append([]string(nil), brief.Warnings...),
	}
	for _, order := range orders.Orders {
		if isScheduleOrder(order) || order.Status != OrderStatusActive {
			continue
		}
		state.ActiveOrders = append(state.ActiveOrders, cloneOrder(order))
	}
	for _, ticket := range brief.Tickets {
		files := append([]string(nil), ticket.Files...)
		sort.Strings(files)
		state.Tickets = append(state.Tickets, scheduleDecisionTicket{
			Target:     ticket.Target,
			TargetType: string(ticket.TargetType),
			CookID:     ticket.CookID,
			Files:      files,
			Status:     string(ticket.Status),
			BlockedBy:  ticket.BlockedBy,
			Reason:     ticket.Reason,
		})
	}
	latest, err := latestScheduleDecisionEvent(filepath.Join(l.runtimeDir, "loop-events.ndjson"))
	if err != nil {
		return "", err
	}
	state.LatestEvent = latest

	sort.Slice(state.Backlog, func(i, j int) bool {
		if state.Backlog[i].ID != state.Backlog[j].ID {
			return state.Backlog[i].ID < state.Backlog[j].ID
		}
		left, _ := json.Marshal(state.Backlog[i])
		right, _ := json.Marshal(state.Backlog[j])
		return string(left) < string(right)
	})
	sort.Slice(state.ActiveOrders, func(i, j int) bool { return state.ActiveOrders[i].ID < state.ActiveOrders[j].ID })
	sort.Strings(state.ActionNeeded)
	sort.Slice(state.Tickets, func(i, j int) bool {
		if state.Tickets[i].Target != state.Tickets[j].Target {
			return state.Tickets[i].Target < state.Tickets[j].Target
		}
		return state.Tickets[i].CookID < state.Tickets[j].CookID
	})
	sort.Strings(state.Routing.AvailableRuntimes)
	sort.Slice(state.TaskTypes, func(i, j int) bool { return state.TaskTypes[i].Key < state.TaskTypes[j].Key })
	sort.Strings(state.Warnings)

	payload, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("encode schedule decision state: %w", err)
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

func latestScheduleDecisionEvent(path string) (*scheduleDecisionEvent, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read schedule decision events: %w", err)
	}
	defer f.Close()

	var latest *scheduleDecisionEvent
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry event.LoopEvent
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if !scheduleDecisionEventRelevant(entry.Type) {
			continue
		}
		candidate := scheduleDecisionEvent{
			Seq:     entry.Seq,
			Type:    string(entry.Type),
			Summary: scheduleDecisionEventSummary(entry.Payload),
		}
		if latest == nil || candidate.Seq > latest.Seq {
			latest = &candidate
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan schedule decision events: %w", err)
	}
	return latest, nil
}

func scheduleDecisionEventRelevant(eventType event.LoopEventType) bool {
	switch eventType {
	case event.LoopEventStageCompleted,
		event.LoopEventOrderCompleted,
		event.LoopEventWorktreeMerged,
		event.LoopEventScheduleCompleted,
		event.LoopEventRegistryRebuilt,
		event.LoopEventBootstrapCompleted:
		return false
	default:
		return true
	}
}

func scheduleDecisionEventSummary(payload json.RawMessage) string {
	if len(payload) == 0 {
		return ""
	}
	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		return ""
	}
	normalized, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(normalized)
}

func (l *Loop) writeScheduleEmptyMemo(brief mise.Brief, orders OrdersFile) error {
	digest, err := l.scheduleDecisionDigest(brief, orders)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(scheduleEmptyMemo{Version: scheduleEmptyMemoVersion, Digest: digest})
	if err != nil {
		return fmt.Errorf("encode empty schedule memo: %w", err)
	}
	if err := filex.WriteFileAtomic(l.scheduleEmptyMemoPath(), append(payload, '\n')); err != nil {
		return fmt.Errorf("write empty schedule memo: %w", err)
	}
	return nil
}

func (l *Loop) readScheduleEmptyMemo() (scheduleEmptyMemo, bool, error) {
	data, err := os.ReadFile(l.scheduleEmptyMemoPath())
	if os.IsNotExist(err) {
		return scheduleEmptyMemo{}, false, nil
	}
	if err != nil {
		return scheduleEmptyMemo{}, false, fmt.Errorf("read empty schedule memo: %w", err)
	}
	var memo scheduleEmptyMemo
	if err := json.Unmarshal(data, &memo); err != nil {
		return scheduleEmptyMemo{}, false, fmt.Errorf("parse empty schedule memo: %w", err)
	}
	decoded, err := hex.DecodeString(strings.TrimSpace(memo.Digest))
	if memo.Version != scheduleEmptyMemoVersion || err != nil || len(decoded) != sha256.Size {
		return scheduleEmptyMemo{}, false, fmt.Errorf("invalid empty schedule memo")
	}
	return memo, true, nil
}

func (l *Loop) scheduleEmptyMemoMatches(brief mise.Brief, orders OrdersFile) bool {
	memo, exists, err := l.readScheduleEmptyMemo()
	if err != nil {
		l.logger.Warn("empty schedule memo unreadable, allowing scheduler", "error", err)
		return false
	}
	if !exists {
		return false
	}
	digest, err := l.scheduleDecisionDigest(brief, orders)
	if err != nil {
		l.logger.Warn("schedule decision digest unavailable, allowing scheduler", "error", err)
		return false
	}
	return memo.Digest == digest
}

func (l *Loop) clearScheduleEmptyMemo() error {
	err := os.Remove(l.scheduleEmptyMemoPath())
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return fmt.Errorf("remove empty schedule memo: %w", err)
}

package mise

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/event"
)

type Builder struct {
	projectDir  string
	runtimeDir  string
	config      config.Config
	runner      *adapter.Runner
	now         func() time.Time
	TaskTypes   []TaskTypeSummary
	lastContent []byte // JSON sans GeneratedAt for change detection
}

func NewBuilder(projectDir string, cfg config.Config) *Builder {
	projectDir = strings.TrimSpace(projectDir)
	return &Builder{
		projectDir: projectDir,
		runtimeDir: filepath.Join(projectDir, ".noodle"),
		config:     cfg,
		runner:     adapter.NewRunner(projectDir, cfg),
		now:        time.Now,
	}
}

func (b *Builder) Build(ctx context.Context, activeSummary ActiveSummary, recentHistory []HistoryItem) (Brief, []string, bool, error) {
	warnings := make([]string, 0)
	backlog := make([]adapter.BacklogItem, 0)

	if _, ok := b.config.Adapters["backlog"]; ok {
		if strings.TrimSpace(b.config.Adapters["backlog"].Scripts["sync"]) == "" {
			warnings = append(warnings, "backlog sync script missing; returning empty backlog")
		} else {
			items, err := b.runner.SyncBacklog(ctx)
			if err != nil {
				if isMissingSyncScriptError(err) {
					warnings = append(warnings, "backlog sync script missing; returning empty backlog")
				} else {
					return Brief{}, warnings, false, err
				}
			} else {
				backlog = filterActiveBacklog(items)
			}
		}
	}

	tickets, err := b.readTickets()
	if err != nil {
		return Brief{}, warnings, false, err
	}

	if activeSummary.ByTaskKey == nil {
		activeSummary.ByTaskKey = map[string]int{}
	}
	if activeSummary.ByStatus == nil {
		activeSummary.ByStatus = map[string]int{}
	}
	if activeSummary.ByRuntime == nil {
		activeSummary.ByRuntime = map[string]int{}
	}
	if recentHistory == nil {
		recentHistory = []HistoryItem{}
	}

	resources := ResourceSnapshot{
		MaxCooks: b.config.Concurrency.MaxCooks,
		Active:   activeSummary.Total,
	}
	resources.Available = resources.MaxCooks - resources.Active
	if resources.Available < 0 {
		resources.Available = 0
	}

	routing := RoutingSnapshot{
		Defaults:          routingPolicyFromModelPolicy(b.config.Routing.Defaults),
		Tags:              make(map[string]RoutingPolicy, len(b.config.Routing.Tags)),
		AvailableRuntimes: b.config.AvailableRuntimes(),
	}
	for tag, policy := range b.config.Routing.Tags {
		routing.Tags[tag] = routingPolicyFromModelPolicy(policy)
	}

	recentEvents := readRecentEvents(b.runtimeDir)

	brief := Brief{
		GeneratedAt:   b.now().UTC(),
		Backlog:       backlog,
		ActiveSummary: activeSummary,
		Tickets:       tickets,
		Resources:     resources,
		RecentHistory: recentHistory,
		RecentEvents:  recentEvents,
		Routing:       routing,
		TaskTypes:     b.TaskTypes,
		Warnings:      warnings,
	}

	// Only write when content actually changed (ignore GeneratedAt).
	cmp := brief
	cmp.GeneratedAt = time.Time{}
	content, err := json.Marshal(cmp)
	if err != nil {
		return Brief{}, warnings, false, fmt.Errorf("encode mise json for comparison: %w", err)
	}
	changed := !bytes.Equal(content, b.lastContent)
	if changed {
		if err := writeBriefAtomic(filepath.Join(b.runtimeDir, "mise.json"), brief); err != nil {
			return Brief{}, warnings, false, err
		}
		b.lastContent = content
	}
	return brief, warnings, changed, nil
}

func routingPolicyFromModelPolicy(policy config.ModelPolicy) RoutingPolicy {
	return RoutingPolicy{
		Provider: policy.Provider,
		Model:    policy.Model,
	}
}

func filterActiveBacklog(items []adapter.BacklogItem) []adapter.BacklogItem {
	filtered := make([]adapter.BacklogItem, 0, len(items))
	for _, item := range items {
		if item.Status == adapter.BacklogStatusDone {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func (b *Builder) readTickets() ([]event.Ticket, error) {
	path := filepath.Join(b.runtimeDir, "tickets.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []event.Ticket{}, nil
		}
		return nil, fmt.Errorf("read tickets.json: %w", err)
	}
	var tickets []event.Ticket
	if err := json.Unmarshal(data, &tickets); err != nil {
		return nil, fmt.Errorf("parse tickets.json: %w", err)
	}
	if tickets == nil {
		return []event.Ticket{}, nil
	}
	return tickets, nil
}

const maxRecentEvents = 50

// readRecentEvents reads loop-events.ndjson and returns events after the last
// schedule.completed watermark. Returns an empty slice on any error.
func readRecentEvents(runtimeDir string) []RecentEvent {
	path := filepath.Join(runtimeDir, "loop-events.ndjson")
	f, err := os.Open(path)
	if err != nil {
		return []RecentEvent{}
	}
	defer f.Close()

	// Minimal struct for unmarshalling NDJSON lines — no import cycle.
	type ndjsonLine struct {
		Seq     uint64          `json:"seq"`
		Type    string          `json:"type"`
		At      time.Time       `json:"at"`
		Payload json.RawMessage `json:"payload,omitempty"`
	}

	var lines []ndjsonLine
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		var line ndjsonLine
		if err := json.Unmarshal([]byte(raw), &line); err != nil {
			continue
		}
		lines = append(lines, line)
	}

	// Find watermark: seq of the last schedule.completed event.
	var watermark uint64
	for _, line := range lines {
		if line.Type == "schedule.completed" && line.Seq > watermark {
			watermark = line.Seq
		}
	}

	// Collect events after watermark.
	var result []RecentEvent
	for _, line := range lines {
		if line.Seq <= watermark {
			continue
		}
		result = append(result, RecentEvent{
			Type:    line.Type,
			Seq:     line.Seq,
			At:      line.At,
			Summary: eventSummary(line.Type, line.Payload),
		})
	}

	// Cap at most recent.
	if len(result) > maxRecentEvents {
		result = result[len(result)-maxRecentEvents:]
	}
	return result
}

// eventSummary derives a human-readable summary from the event type and payload.
func eventSummary(eventType string, payload json.RawMessage) string {
	var fields map[string]json.RawMessage
	if len(payload) > 0 {
		_ = json.Unmarshal(payload, &fields)
	}

	getString := func(key string) string {
		raw, ok := fields[key]
		if !ok {
			return ""
		}
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return ""
		}
		return s
	}
	getNestedString := func(parent string, key string) string {
		raw, ok := fields[parent]
		if !ok {
			return ""
		}
		var nested map[string]json.RawMessage
		if err := json.Unmarshal(raw, &nested); err != nil {
			return ""
		}
		value, ok := nested[key]
		if !ok {
			return ""
		}
		var s string
		if err := json.Unmarshal(value, &s); err != nil {
			return ""
		}
		return s
	}

	orderID := getString("order_id")
	reason := getString("reason")
	owner := getNestedString("failure", "owner")
	failureClass := getNestedString("failure", "class")
	prefix := ""
	if owner != "" && failureClass == "agent_mistake" {
		prefix = "[" + owner + "] "
	}

	switch eventType {
	case "stage.completed":
		taskKey := getString("task_key")
		if taskKey != "" {
			return taskKey + " stage completed for " + orderID
		}
		return "stage completed for " + orderID
	case "stage.failed":
		if reason != "" {
			return prefix + "stage failed for " + orderID + ": " + reason
		}
		return prefix + "stage failed for " + orderID
	case "order.completed":
		return "order " + orderID + " completed"
	case "order.failed":
		if reason != "" {
			return prefix + "order " + orderID + " failed: " + reason
		}
		return prefix + "order " + orderID + " failed"
	case "promotion.failed":
		if reason != "" {
			return prefix + "orders-next rejected: " + reason
		}
		return prefix + "orders-next rejected"
	case "order.dropped":
		return "order " + orderID + " dropped"
	case "order.requeued":
		return "order " + orderID + " requeued"
	case "worktree.merged":
		return "worktree merged for " + orderID
	case "merge.conflict":
		return "merge conflict for " + orderID
	case "registry.rebuilt":
		return "skill registry rebuilt"
	case "sync.degraded":
		if reason != "" {
			return "sync degraded: " + reason
		}
		return "sync degraded"
	case "bootstrap.completed":
		return "bootstrap completed"
	case "bootstrap.exhausted":
		return "bootstrap exhausted"
	default:
		return eventType
	}
}

func isMissingSyncScriptError(err error) bool {
	raw := strings.ToLower(err.Error())
	switch {
	case strings.Contains(raw, "script is not configured"):
		return true
	case strings.Contains(raw, " not found"):
		return true
	case strings.Contains(raw, "no such file"):
		return true
	default:
		return false
	}
}

package snapshot

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/poteto/noodle/loop"
)

func readSteerEvents(controlPath string) []FeedEvent {
	file, err := os.Open(controlPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var events []FeedEvent
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var cmd loop.ControlCommand
		if err := json.Unmarshal([]byte(line), &cmd); err != nil {
			continue
		}
		if !strings.EqualFold(cmd.Action, "steer") {
			continue
		}
		target := strings.TrimSpace(cmd.Target)
		prompt := strings.TrimSpace(cmd.Prompt)
		if target == "" && prompt == "" {
			continue
		}
		at := cmd.At
		if at.IsZero() {
			at = time.Now().UTC()
		}
		events = append(events, FeedEvent{
			SessionID: "chef",
			AgentName: target,
			At:        at,
			Label:     "Steer",
			Body:      prompt,
			Category:  "steer",
		})
	}
	return events
}

func readLoopEvents(runtimeDir string) []FeedEvent {
	path := filepath.Join(runtimeDir, "loop-events.ndjson")
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var events []FeedEvent
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw struct {
			Seq     uint64          `json:"seq"`
			Type    string          `json:"type"`
			At      time.Time       `json:"at"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}

		var (
			label, body, category, taskType string
			failureMetadata                 *loop.EventFailureMetadata
		)
		switch raw.Type {
		case "stage.completed":
			label = "Completed"
			category = "stage_completed"
			var p struct {
				OrderID string  `json:"order_id"`
				TaskKey string  `json:"task_key"`
				Message *string `json:"message"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			taskType = p.TaskKey
			if p.Message != nil && *p.Message != "" {
				body = *p.Message
			}
		case "stage.failed":
			label = "Failed"
			category = "stage_failed"
			var p struct {
				OrderID string                     `json:"order_id"`
				TaskKey string                     `json:"task_key"`
				Reason  string                     `json:"reason"`
				Failure *loop.EventFailureMetadata `json:"failure"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			failureMetadata = p.Failure
			taskType = p.TaskKey
			body = p.Reason
			if owner := failureOwnerPrefix(p.Failure); owner != "" && body != "" {
				body = owner + body
			}
		case "order.completed":
			label = "Order Complete"
			category = "order_completed"
			var p struct {
				OrderID string `json:"order_id"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			body = p.OrderID
		case "order.failed":
			label = "Order Failed"
			category = "order_failed"
			var p struct {
				OrderID string                     `json:"order_id"`
				Reason  string                     `json:"reason"`
				Failure *loop.EventFailureMetadata `json:"failure"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			failureMetadata = p.Failure
			if p.Reason != "" {
				body = p.Reason
			} else {
				body = p.OrderID
			}
			if owner := failureOwnerPrefix(p.Failure); owner != "" && body != "" {
				body = owner + body
			}
		case "order.dropped":
			label = "Dropped"
			category = "order_drop"
			var p struct {
				OrderID string `json:"order_id"`
				Reason  string `json:"reason"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			if p.Reason != "" {
				body = fmt.Sprintf("Dropped order %s: %s", p.OrderID, p.Reason)
			} else {
				body = fmt.Sprintf("Dropped order %s", p.OrderID)
			}
		case "order.requeued":
			label = "Requeued"
			category = "order_requeued"
			var p struct {
				OrderID string `json:"order_id"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			body = p.OrderID
		case "worktree.merged":
			label = "Merged"
			category = "worktree_merged"
			var p struct {
				OrderID      string `json:"order_id"`
				WorktreeName string `json:"worktree_name"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			body = p.WorktreeName
		case "merge.conflict":
			label = "Conflict"
			category = "merge_conflict"
			var p struct {
				OrderID      string `json:"order_id"`
				WorktreeName string `json:"worktree_name"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			body = p.WorktreeName
		case "schedule.completed":
			label = "Scheduled"
			category = "schedule_completed"
			taskType = "schedule"
		case "registry.rebuilt":
			label = "Rebuild"
			category = "registry_rebuild"
			var p struct {
				Added   []string `json:"added"`
				Removed []string `json:"removed"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			body = fmt.Sprintf("Registry rebuilt — added: %v, removed: %v", p.Added, p.Removed)
		case "bootstrap.completed":
			label = "Bootstrap"
			category = "bootstrap"
			body = "Bootstrap completed"
		case "bootstrap.exhausted":
			label = "Bootstrap"
			category = "bootstrap"
			var p struct {
				Reason  string                     `json:"reason"`
				Failure *loop.EventFailureMetadata `json:"failure"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			failureMetadata = p.Failure
			if p.Reason != "" {
				body = p.Reason
			} else {
				body = "Bootstrap exhausted"
			}
		case "sync.degraded":
			label = "Sync"
			category = "sync_degraded"
			var p struct {
				Reason  string                     `json:"reason"`
				Failure *loop.EventFailureMetadata `json:"failure"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			failureMetadata = p.Failure
			if p.Reason != "" {
				body = p.Reason
			} else {
				body = "Sync degraded"
			}
		case "promotion.failed":
			label = "Rejected"
			category = "promotion_failed"
			var p struct {
				Reason  string                     `json:"reason"`
				Failure *loop.EventFailureMetadata `json:"failure"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			failureMetadata = p.Failure
			if p.Reason != "" {
				body = p.Reason
			} else {
				body = "orders-next.json validation failed"
			}
			if owner := failureOwnerPrefix(p.Failure); owner != "" && body != "" {
				body = owner + body
			}
		default:
			continue
		}

		at := raw.At
		if at.IsZero() {
			at = time.Now().UTC()
		}
		events = append(events, FeedEvent{
			SessionID: "loop",
			AgentName: "loop",
			TaskType:  taskType,
			At:        at,
			Label:     label,
			Body:      body,
			Category:  category,
			Failure:   failureMetadata,
		})
	}
	return events
}

func failureOwnerPrefix(metadata *loop.EventFailureMetadata) string {
	if metadata == nil || metadata.Owner == "" {
		return ""
	}
	return "[" + string(metadata.Owner) + "] "
}

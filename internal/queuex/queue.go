package queuex

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/filex"
	"github.com/poteto/noodle/internal/stringx"
	"github.com/poteto/noodle/internal/taskreg"
)

// ErrUnknownTaskType is returned when a queue item references a task type
// not present in the registry.
var ErrUnknownTaskType = errors.New("unknown task type")

// Stage status constants.
const (
	StageStatusPending   = "pending"
	StageStatusActive    = "active"
	StageStatusCompleted = "completed"
	StageStatusFailed    = "failed"
	StageStatusCancelled = "cancelled"
)

// Order status constants.
const (
	OrderStatusActive    = "active"
	OrderStatusCompleted = "completed"
	OrderStatusFailed    = "failed"
	OrderStatusFailing   = "failing"
)

// Stage is a unit of work within an order (serialization type).
type Stage struct {
	TaskKey  string                     `json:"task_key,omitempty"`
	Prompt   string                     `json:"prompt,omitempty"`
	Skill    string                     `json:"skill,omitempty"`
	Provider string                     `json:"provider"`
	Model    string                     `json:"model"`
	Runtime  string                     `json:"runtime,omitempty"`
	Status   string                     `json:"status"`
	Extra    map[string]json.RawMessage `json:"extra,omitempty"`
}

// Order is a pipeline of stages (serialization type).
type Order struct {
	ID        string   `json:"id"`
	Title     string   `json:"title,omitempty"`
	Plan      []string `json:"plan,omitempty"`
	Rationale string   `json:"rationale,omitempty"`
	Stages    []Stage  `json:"stages"`
	Status    string   `json:"status"`
	OnFailure []Stage  `json:"on_failure,omitempty"`
}

// OrdersFile is the top-level orders.json structure (serialization type).
type OrdersFile struct {
	GeneratedAt  time.Time `json:"generated_at"`
	Orders       []Order   `json:"orders"`
	ActionNeeded []string  `json:"action_needed,omitempty"`
}

// ValidateOrderStatus returns an error if the order status is not valid.
func ValidateOrderStatus(status string) error {
	switch status {
	case OrderStatusActive, OrderStatusCompleted, OrderStatusFailed, OrderStatusFailing:
		return nil
	case "":
		return fmt.Errorf("order status is required")
	default:
		return fmt.Errorf("unknown order status %q", status)
	}
}

// ValidateStageStatus returns an error if the stage status is not valid.
func ValidateStageStatus(status string) error {
	switch status {
	case StageStatusPending, StageStatusActive, StageStatusCompleted, StageStatusFailed, StageStatusCancelled:
		return nil
	case "":
		return fmt.Errorf("stage status is required")
	default:
		return fmt.Errorf("unknown stage status %q", status)
	}
}

const scheduleTaskKey = "schedule"

// Queue is the canonical queue.json contract.
type Queue struct {
	GeneratedAt  time.Time `json:"generated_at"`
	Items        []Item    `json:"items"`
	ActionNeeded []string  `json:"action_needed,omitempty"`
}

// Item is one queue entry.
type Item struct {
	ID        string   `json:"id"`
	TaskKey   string   `json:"task_key,omitempty"`
	Title     string   `json:"title,omitempty"`
	Prompt    string   `json:"prompt,omitempty"`
	Provider  string   `json:"provider"`
	Model     string   `json:"model"`
	Runtime   string   `json:"runtime,omitempty"`
	Skill     string   `json:"skill,omitempty"`
	Plan      []string `json:"plan,omitempty"`
	Rationale string   `json:"rationale,omitempty"`
}

func Read(path string) (Queue, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Queue{}, nil
		}
		return Queue{}, fmt.Errorf("read queue: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return Queue{}, nil
	}
	q, err := decodeQueue(data, true)
	if err != nil {
		// Corrupted queue — treat as empty rather than blocking the loop.
		return Queue{}, nil
	}
	return q, nil
}

// ParseStrict validates queue bytes without reading from disk.
func ParseStrict(data []byte) (Queue, error) {
	if strings.TrimSpace(string(data)) == "" {
		return Queue{}, nil
	}
	return decodeQueue(data, false)
}

// ReadStrict parses only the canonical wrapped queue object and rejects legacy arrays.
func ReadStrict(path string) (Queue, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Queue{}, nil
		}
		return Queue{}, fmt.Errorf("read queue: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return Queue{}, nil
	}
	return decodeQueue(data, false)
}

func decodeQueue(data []byte, allowLegacyArray bool) (Queue, error) {
	var wrapped Queue
	if err := json.Unmarshal(data, &wrapped); err == nil {
		if wrapped.Items == nil {
			wrapped.Items = []Item{}
		}
		return wrapped, nil
	}

	// Legacy compatibility: queue can be a bare array.
	if allowLegacyArray {
		var items []Item
		if err := json.Unmarshal(data, &items); err == nil {
			return Queue{Items: items}, nil
		}
	}
	return Queue{}, fmt.Errorf("parse queue: invalid JSON")
}

func WriteAtomic(path string, queue Queue) error {
	data, err := json.MarshalIndent(queue, "", "  ")
	if err != nil {
		return fmt.Errorf("encode queue: %w", err)
	}
	if err := filex.WriteFileAtomic(path, append(data, '\n')); err != nil {
		return fmt.Errorf("rename queue file: %w", err)
	}
	return nil
}

func ApplyRoutingDefaults(queue Queue, reg taskreg.Registry, cfg config.Config) (Queue, bool) {
	items := make([]Item, len(queue.Items))
	copy(items, queue.Items)
	changed := false
	for i := range items {
		updated, itemChanged := applyItemRoutingDefaults(items[i], reg, cfg)
		if itemChanged {
			changed = true
			items[i] = updated
		}
	}
	if !changed {
		return queue, false
	}
	queue.Items = items
	return queue, true
}

func NormalizeAndValidate(
	queue Queue,
	schedulablePlanIDs []int,
	reg taskreg.Registry,
	cfg config.Config,
) (Queue, bool, error) {
	schedulableSet := make(map[int]struct{}, len(schedulablePlanIDs))
	for _, id := range schedulablePlanIDs {
		schedulableSet[id] = struct{}{}
	}

	items := make([]Item, len(queue.Items))
	copy(items, queue.Items)
	changed := false
	seenIDs := make(map[string]struct{}, len(items))
	kept := make([]Item, 0, len(items))
	for i := range items {
		id := strings.TrimSpace(items[i].ID)
		if id == "" {
			return queue, false, fmt.Errorf("queue item id is required")
		}
		if _, exists := seenIDs[id]; exists {
			return queue, false, fmt.Errorf("queue item %q appears more than once", id)
		}
		seenIDs[id] = struct{}{}

		taskType, ok := reg.ResolveQueueItem(taskreg.QueueItemInput{
			ID:      items[i].ID,
			TaskKey: items[i].TaskKey,
			Title:   items[i].Title,
			Skill:   items[i].Skill,
		})
		if !ok && isScheduleBootstrapItem(items[i]) {
			taskType = taskreg.TaskType{Key: scheduleTaskKey}
			ok = true
		}
		if !ok {
			return queue, false, fmt.Errorf("queue item %q: %w", id, ErrUnknownTaskType)
		}

		if strings.TrimSpace(items[i].TaskKey) != taskType.Key {
			items[i].TaskKey = taskType.Key
			changed = true
		}
		if strings.TrimSpace(items[i].Skill) == "" {
			items[i].Skill = taskType.Key
			changed = true
		}
		// Execute items with numeric IDs must map to a schedulable plan.
		// Non-numeric IDs (e.g. "execute-1771969840249" from the task creator)
		// are ad-hoc tasks that don't require a plan.
		// If the plan was deleted, drop the orphaned item.
		if taskType.Key == "execute" && len(schedulableSet) > 0 {
			if planID, parseErr := strconv.Atoi(id); parseErr == nil {
				if _, exists := schedulableSet[planID]; !exists {
					changed = true
					continue
				}
			}
		}
		kept = append(kept, items[i])
	}
	items = kept

	if !changed {
		return queue, false, nil
	}
	queue.Items = items
	return queue, true, nil
}

func isScheduleBootstrapItem(item Item) bool {
	id := strings.ToLower(strings.TrimSpace(item.ID))
	taskKey := strings.ToLower(strings.TrimSpace(item.TaskKey))
	skill := strings.ToLower(strings.TrimSpace(item.Skill))
	return id == scheduleTaskKey || taskKey == scheduleTaskKey || skill == scheduleTaskKey
}

func applyItemRoutingDefaults(item Item, reg taskreg.Registry, cfg config.Config) (Item, bool) {
	changed := false
	defaultProvider := strings.TrimSpace(cfg.Routing.Defaults.Provider)
	defaultModel := strings.TrimSpace(cfg.Routing.Defaults.Model)
	tagProvider := ""
	tagModel := ""

	if taskType, ok := reg.ResolveQueueItem(taskreg.QueueItemInput{
		ID:      item.ID,
		TaskKey: item.TaskKey,
		Title:   item.Title,
		Skill:   item.Skill,
	}); ok {
		if policy, exists := cfg.Routing.Tags[taskType.Key]; exists {
			tagProvider = strings.TrimSpace(policy.Provider)
			tagModel = strings.TrimSpace(policy.Model)
		}
	}

	if strings.TrimSpace(item.Provider) == "" {
		provider := stringx.FirstNonEmpty(tagProvider, defaultProvider)
		if provider != "" {
			item.Provider = provider
			changed = true
		}
	}
	if strings.TrimSpace(item.Model) == "" {
		model := stringx.FirstNonEmpty(tagModel, defaultModel)
		if model != "" {
			item.Model = model
			changed = true
		}
	}
	return item, changed
}

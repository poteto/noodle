package queuex

import (
	"encoding/json"
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

const prioritizeTaskKey = "prioritize"

// Queue is the canonical queue.json contract.
type Queue struct {
	GeneratedAt  time.Time `json:"generated_at"`
	Items        []Item    `json:"items"`
	Active       []string  `json:"active,omitempty"`
	ActionNeeded []string  `json:"action_needed,omitempty"`
	Autonomy     string    `json:"autonomy,omitempty"`
	LoopState    string    `json:"loop_state,omitempty"`
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
	Review    *bool    `json:"review,omitempty"`
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
	return decodeQueue(data, true)
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
		if !ok && strings.TrimSpace(items[i].TaskKey) == "" && strings.TrimSpace(items[i].Skill) == "" {
			if resolved, found := reg.ByKey("execute"); found {
				taskType = resolved
				ok = true
			}
		}
		if !ok && isPrioritizeBootstrapItem(items[i]) {
			taskType = taskreg.TaskType{Key: prioritizeTaskKey}
			ok = true
		}
		if !ok {
			return queue, false, fmt.Errorf("queue item %q has unknown task type", id)
		}

		if strings.TrimSpace(items[i].TaskKey) != taskType.Key {
			items[i].TaskKey = taskType.Key
			changed = true
		}
		if strings.TrimSpace(items[i].Skill) == "" {
			items[i].Skill = taskType.Key
			changed = true
		}
		// Execute items must map to a schedulable plan ID when available.
		if taskType.Key == "execute" && len(schedulableSet) > 0 {
			planID, parseErr := strconv.Atoi(id)
			if parseErr != nil {
				return queue, false, fmt.Errorf(
					"queue item %q is an execute task but does not match a schedulable plan",
					id,
				)
			}
			if _, exists := schedulableSet[planID]; !exists {
				return queue, false, fmt.Errorf(
					"queue item %q is an execute task but does not match a schedulable plan",
					id,
				)
			}
		}
	}

	if !changed {
		return queue, false, nil
	}
	queue.Items = items
	return queue, true, nil
}

func isPrioritizeBootstrapItem(item Item) bool {
	id := strings.ToLower(strings.TrimSpace(item.ID))
	taskKey := strings.ToLower(strings.TrimSpace(item.TaskKey))
	skill := strings.ToLower(strings.TrimSpace(item.Skill))
	return id == prioritizeTaskKey || taskKey == prioritizeTaskKey || skill == prioritizeTaskKey
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

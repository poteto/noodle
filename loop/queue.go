package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
)

func readQueue(path string) (Queue, error) {
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
	var queue Queue
	if err := json.Unmarshal(data, &queue); err != nil {
		return Queue{}, fmt.Errorf("parse queue: %w", err)
	}
	return queue, nil
}

func writeQueueAtomic(path string, queue Queue) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create queue parent directory: %w", err)
	}
	data, err := json.MarshalIndent(queue, "", "  ")
	if err != nil {
		return fmt.Errorf("encode queue: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write queue temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename queue file: %w", err)
	}
	return nil
}

func applyQueueRoutingDefaults(queue Queue, cfg config.Config) (Queue, bool) {
	items := make([]QueueItem, len(queue.Items))
	copy(items, queue.Items)
	changed := false
	for i := range items {
		items[i], changed = applyQueueItemRoutingDefaults(items[i], cfg, changed)
	}
	if !changed {
		return queue, false
	}
	queue.Items = items
	return queue, true
}

func applyQueueItemRoutingDefaults(item QueueItem, cfg config.Config, changed bool) (QueueItem, bool) {
	defaultProvider := strings.TrimSpace(cfg.Routing.Defaults.Provider)
	defaultModel := strings.TrimSpace(cfg.Routing.Defaults.Model)
	tagProvider := ""
	tagModel := ""
	if taskType, ok := taskTypeForQueueItem(cfg, item); ok {
		if policy, ok := cfg.Routing.Tags[taskType.Key]; ok {
			tagProvider = strings.TrimSpace(policy.Provider)
			tagModel = strings.TrimSpace(policy.Model)
		}
	}

	if strings.TrimSpace(item.Provider) == "" {
		provider := nonEmpty(tagProvider, defaultProvider)
		if provider != "" {
			item.Provider = provider
			changed = true
		}
	}
	if strings.TrimSpace(item.Model) == "" {
		model := nonEmpty(tagModel, defaultModel)
		if model != "" {
			item.Model = model
			changed = true
		}
	}
	return item, changed
}

func normalizeAndValidateQueue(queue Queue, backlog []adapter.BacklogItem, cfg config.Config) (Queue, bool, error) {
	backlogIDs := make(map[string]struct{}, len(backlog))
	for _, item := range backlog {
		if item.Status == adapter.BacklogStatusDone {
			continue
		}
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		backlogIDs[id] = struct{}{}
	}

	items := make([]QueueItem, len(queue.Items))
	copy(items, queue.Items)
	changed := false
	for i := range items {
		id := strings.TrimSpace(items[i].ID)
		if id == "" {
			return queue, false, fmt.Errorf("queue item id is required")
		}

		taskType, ok := taskTypeForQueueItem(cfg, items[i])
		if !ok {
			if strings.TrimSpace(items[i].TaskKey) == "" && strings.TrimSpace(items[i].Skill) == "" {
				if resolved, found := configuredTaskTypeByKey(cfg, executeTaskKey()); found {
					taskType = resolved
					ok = true
				}
			}
		}
		if !ok {
			return queue, false, fmt.Errorf("queue item %q has unknown task type", id)
		}

		if strings.TrimSpace(items[i].TaskKey) != taskType.Key {
			items[i].TaskKey = taskType.Key
			changed = true
		}
		if strings.TrimSpace(items[i].Skill) == "" && strings.TrimSpace(taskType.Skill) != "" {
			items[i].Skill = taskType.Skill
			changed = true
		}
		if !taskType.Synthetic && len(backlogIDs) > 0 {
			if _, exists := backlogIDs[id]; !exists {
				return queue, false, fmt.Errorf(
					"queue item %q uses non-synthetic task type %q but is not in backlog",
					id,
					taskType.Key,
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

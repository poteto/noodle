package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

func queueFromBacklog(items []adapter.BacklogItem, cfg config.Config) Queue {
	queue := Queue{Items: make([]QueueItem, 0, len(items))}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	for _, item := range items {
		if item.Status == adapter.BacklogStatusDone {
			continue
		}
		queue.Items = append(queue.Items, QueueItem{
			ID:       item.ID,
			Title:    item.Title,
			Provider: cfg.Routing.Defaults.Provider,
			Model:    cfg.Routing.Defaults.Model,
		})
	}
	return queue
}

func applyQueueRoutingDefaults(queue Queue, cfg config.Config) (Queue, bool) {
	provider := strings.TrimSpace(cfg.Routing.Defaults.Provider)
	model := strings.TrimSpace(cfg.Routing.Defaults.Model)
	if provider == "" && model == "" {
		return queue, false
	}

	items := make([]QueueItem, len(queue.Items))
	copy(items, queue.Items)
	changed := false
	for i := range items {
		if provider != "" && strings.TrimSpace(items[i].Provider) != provider {
			items[i].Provider = provider
			changed = true
		}
		if model != "" && strings.TrimSpace(items[i].Model) != model {
			items[i].Model = model
			changed = true
		}
	}
	if !changed {
		return queue, false
	}
	queue.Items = items
	return queue, true
}

package loop

import (
	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/queuex"
	"github.com/poteto/noodle/internal/taskreg"
)

func readQueue(path string) (Queue, error) {
	queue, err := queuex.Read(path)
	if err != nil {
		return Queue{}, err
	}
	return fromQueueX(queue), nil
}

func writeQueueAtomic(path string, queue Queue) error {
	return queuex.WriteAtomic(path, toQueueX(queue))
}

func applyQueueRoutingDefaults(queue Queue, cfg config.Config) (Queue, bool) {
	reg := taskreg.New(cfg)
	updated, changed := queuex.ApplyRoutingDefaults(toQueueX(queue), reg, cfg)
	if !changed {
		return queue, false
	}
	return fromQueueX(updated), true
}

func normalizeAndValidateQueue(queue Queue, backlog []adapter.BacklogItem, cfg config.Config) (Queue, bool, error) {
	reg := taskreg.New(cfg)
	updated, changed, err := queuex.NormalizeAndValidate(toQueueX(queue), backlog, reg, cfg)
	if err != nil {
		return Queue{}, false, err
	}
	if !changed {
		return queue, false, nil
	}
	return fromQueueX(updated), true, nil
}

func toQueueX(queue Queue) queuex.Queue {
	items := make([]queuex.Item, 0, len(queue.Items))
	for _, item := range queue.Items {
		items = append(items, queuex.Item{
			ID:        item.ID,
			TaskKey:   item.TaskKey,
			Title:     item.Title,
			Provider:  item.Provider,
			Model:     item.Model,
			Skill:     item.Skill,
			Review:    item.Review,
			Rationale: item.Rationale,
		})
	}
	return queuex.Queue{GeneratedAt: queue.GeneratedAt, Items: items}
}

func fromQueueX(queue queuex.Queue) Queue {
	items := make([]QueueItem, 0, len(queue.Items))
	for _, item := range queue.Items {
		items = append(items, QueueItem{
			ID:        item.ID,
			TaskKey:   item.TaskKey,
			Title:     item.Title,
			Provider:  item.Provider,
			Model:     item.Model,
			Skill:     item.Skill,
			Review:    item.Review,
			Rationale: item.Rationale,
		})
	}
	return Queue{GeneratedAt: queue.GeneratedAt, Items: items}
}

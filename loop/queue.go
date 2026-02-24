package loop

import (
	"fmt"
	"os"
	"sort"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/queuex"
	"github.com/poteto/noodle/internal/taskreg"
)

// consumeQueueNext atomically promotes queue-next.json to queue.json.
// Prioritize sessions write to queue-next.json so they never race with
// loop state stamps on queue.json. The loop is the single writer of
// queue.json — this function is the handoff point.
func consumeQueueNext(nextPath, queuePath string) error {
	if _, err := os.Stat(nextPath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("stat queue-next: %w", err)
	}
	// Validate the proposal before promoting it.
	if _, err := queuex.ReadStrict(nextPath); err != nil {
		// Remove invalid proposals so they don't block future cycles.
		_ = os.Remove(nextPath)
		return fmt.Errorf("invalid queue-next.json: %w", err)
	}
	if err := os.Rename(nextPath, queuePath); err != nil {
		return fmt.Errorf("promote queue-next.json: %w", err)
	}
	return nil
}

func readQueue(path string) (Queue, error) {
	queue, err := queuex.ReadStrict(path)
	if err != nil {
		return Queue{}, err
	}
	return fromQueueX(queue), nil
}

func writeQueueAtomic(path string, queue Queue) error {
	return queuex.WriteAtomic(path, toQueueX(queue))
}

func applyQueueRoutingDefaults(queue Queue, reg taskreg.Registry, cfg config.Config) (Queue, bool) {
	updated, changed := queuex.ApplyRoutingDefaults(toQueueX(queue), reg, cfg)
	if !changed {
		return queue, false
	}
	return fromQueueX(updated), true
}

func normalizeAndValidateQueue(queue Queue, schedulablePlanIDs []int, reg taskreg.Registry, cfg config.Config) (Queue, bool, error) {
	updated, changed, err := queuex.NormalizeAndValidate(toQueueX(queue), schedulablePlanIDs, reg, cfg)
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
			Prompt:    item.Prompt,
			Provider:  item.Provider,
			Model:     item.Model,
			Runtime:   item.Runtime,
			Skill:     item.Skill,
			Plan:      item.Plan,
			Review:    item.Review,
			Rationale: item.Rationale,
		})
	}
	return queuex.Queue{GeneratedAt: queue.GeneratedAt, Items: items, Active: queue.Active, ActionNeeded: queue.ActionNeeded, Autonomy: queue.Autonomy, LoopState: queue.LoopState}
}

func fromQueueX(queue queuex.Queue) Queue {
	items := make([]QueueItem, 0, len(queue.Items))
	for _, item := range queue.Items {
		items = append(items, QueueItem{
			ID:        item.ID,
			TaskKey:   item.TaskKey,
			Title:     item.Title,
			Prompt:    item.Prompt,
			Provider:  item.Provider,
			Model:     item.Model,
			Runtime:   item.Runtime,
			Skill:     item.Skill,
			Plan:      item.Plan,
			Review:    item.Review,
			Rationale: item.Rationale,
		})
	}
	return Queue{GeneratedAt: queue.GeneratedAt, Items: items, Active: queue.Active, ActionNeeded: queue.ActionNeeded, Autonomy: queue.Autonomy, LoopState: queue.LoopState}
}

// stampLoopState writes active IDs and autonomy into queue.json so the TUI
// can derive cooking status and display the current autonomy mode.
// Skips the write when nothing changed, avoiding fs-watcher feedback loops.
func (l *Loop) stampLoopState() error {
	queue, err := readQueue(l.deps.QueueFile)
	if err != nil {
		return err
	}

	active := make([]string, 0, len(l.activeByTarget))
	for targetID := range l.activeByTarget {
		active = append(active, targetID)
	}
	sort.Strings(active)

	autonomy := l.config.Autonomy
	loopState := string(l.state)

	// Skip write if nothing changed.
	if slicesEqual(queue.Active, active) && queue.Autonomy == autonomy && queue.LoopState == loopState {
		return nil
	}

	queue.Active = active
	queue.Autonomy = autonomy
	queue.LoopState = loopState
	return writeQueueAtomic(l.deps.QueueFile, queue)
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

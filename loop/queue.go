package loop

import (
	"fmt"
	"os"
	"sort"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/filex"
	"github.com/poteto/noodle/internal/queuex"
	"github.com/poteto/noodle/internal/statusfile"
	"github.com/poteto/noodle/internal/taskreg"
)

// consumeQueueNext atomically promotes queue-next.json to queue.json.
// Prioritize sessions write to queue-next.json so they never race with
// loop state stamps on queue.json. The loop is the single writer of
// queue.json — this function is the handoff point.
//
// Reads bytes once, validates in-memory, then writes to queue.json via
// atomic rename — no TOCTOU window between validate and promote.
func consumeQueueNext(nextPath, queuePath string) error {
	data, err := os.ReadFile(nextPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read queue-next: %w", err)
	}
	// Always remove the proposal so it doesn't block future cycles.
	_ = os.Remove(nextPath)
	// Validate the bytes we already read.
	if _, parseErr := queuex.ParseStrict(data); parseErr != nil {
		return fmt.Errorf("invalid queue-next.json (removed): %w", parseErr)
	}
	if err := filex.WriteFileAtomic(queuePath, data); err != nil {
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
			Rationale: item.Rationale,
		})
	}
	return queuex.Queue{GeneratedAt: queue.GeneratedAt, Items: items, ActionNeeded: queue.ActionNeeded}
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
			Rationale: item.Rationale,
		})
	}
	return Queue{GeneratedAt: queue.GeneratedAt, Items: items, ActionNeeded: queue.ActionNeeded}
}

// stampStatus writes active IDs, loop state, and autonomy into status.json
// so the TUI can derive cooking status and display the current autonomy mode.
// Skips the write when nothing changed, avoiding fs-watcher feedback loops.
func (l *Loop) stampStatus() error {
	active := make([]string, 0, len(l.activeByTarget))
	for targetID := range l.activeByTarget {
		active = append(active, targetID)
	}
	sort.Strings(active)

	status := statusfile.Status{
		Active:    active,
		LoopState: string(l.state),
		Autonomy:  l.config.Autonomy,
	}

	// Skip write if nothing changed.
	if slicesEqual(l.lastStatus.Active, status.Active) &&
		l.lastStatus.LoopState == status.LoopState &&
		l.lastStatus.Autonomy == status.Autonomy {
		return nil
	}

	if err := statusfile.WriteAtomic(l.deps.StatusFile, status); err != nil {
		return err
	}
	l.lastStatus = status
	return nil
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

package loop

import (
	"slices"
	"sort"

	"github.com/poteto/noodle/internal/statusfile"
)

// stampStatus writes active IDs, loop state, and autonomy into status.json
// so the TUI can derive cooking status and display the current autonomy mode.
// Skips the write when nothing changed, avoiding fs-watcher feedback loops.
func (l *Loop) stampStatus() error {
	active := make([]string, 0, len(l.cooks.activeCooksByOrder))
	for targetID := range l.cooks.activeCooksByOrder {
		active = append(active, targetID)
	}
	sort.Strings(active)

	status := statusfile.Status{
		Active:    active,
		LoopState: string(l.state),
		Autonomy:  l.config.Autonomy,
		MaxCooks:  l.config.Concurrency.MaxCooks,
	}

	// Skip write if nothing changed.
	if slices.Equal(l.lastStatus.Active, status.Active) &&
		l.lastStatus.LoopState == status.LoopState &&
		l.lastStatus.Autonomy == status.Autonomy &&
		l.lastStatus.MaxCooks == status.MaxCooks {
		l.publishState()
		return nil
	}

	if err := statusfile.WriteAtomic(l.deps.StatusFile, status); err != nil {
		return err
	}
	l.lastStatus = status
	l.publishState()
	return nil
}


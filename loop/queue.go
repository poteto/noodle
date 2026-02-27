package loop

import (
	"sort"

	"github.com/poteto/noodle/internal/statusfile"
)

// stampStatus writes active IDs, loop state, and autonomy into status.json
// so the TUI can derive cooking status and display the current autonomy mode.
// Skips the write when nothing changed, avoiding fs-watcher feedback loops.
func (l *Loop) stampStatus() error {
	active := make([]string, 0, len(l.activeCooksByOrder))
	for orderID := range l.activeCooksByOrder {
		active = append(active, orderID)
	}
	sort.Strings(active)

	status := statusfile.Status{
		Active:    active,
		LoopState: string(l.state),
		Autonomy:  l.config.Autonomy,
		MaxCooks:  l.config.Concurrency.MaxCooks,
	}

	// Skip write if nothing changed.
	if slicesEqual(l.lastStatus.Active, status.Active) &&
		l.lastStatus.LoopState == status.LoopState &&
		l.lastStatus.Autonomy == status.Autonomy &&
		l.lastStatus.MaxCooks == status.MaxCooks {
		return nil
	}

	if err := statusfile.WriteAtomic(l.deps.StatusFile, status); err != nil {
		return err
	}
	l.lastStatus = status
	l.publishState()
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

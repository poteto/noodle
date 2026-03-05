package loop

import (
	"slices"
	"sort"

	"github.com/poteto/noodle/internal/statusfile"
)

// stampStatus writes active IDs, loop state, and mode into status.json
// so the TUI can derive cooking status and display the current run mode.
// Skips the write when nothing changed, avoiding fs-watcher feedback loops.
func (l *Loop) stampStatus() error {
	active := make([]string, 0, len(l.cooks.activeCooksByOrder))
	for targetID := range l.cooks.activeCooksByOrder {
		active = append(active, targetID)
	}
	sort.Strings(active)

	// Read mode from canonical state (V2 source of truth).
	mode := string(l.canonical.Mode)
	if mode == "" {
		mode = l.config.Mode
	}

	status := statusfile.Status{
		Active:         active,
		LoopState:      string(l.state),
		Mode:           mode,
		MaxConcurrency: l.config.Concurrency.MaxConcurrency,
		Warnings:       l.lastMiseWarnings,
	}

	// Skip write if nothing changed.
	if slices.Equal(l.lastStatus.Active, status.Active) &&
		l.lastStatus.LoopState == status.LoopState &&
		l.lastStatus.Mode == status.Mode &&
		l.lastStatus.MaxConcurrency == status.MaxConcurrency &&
		slices.Equal(l.lastStatus.Warnings, status.Warnings) {
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

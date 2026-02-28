package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/poteto/noodle/cmdmeta"
	"github.com/poteto/noodle/internal/lockfile"
	"github.com/spf13/cobra"
)

func newResetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: cmdmeta.Short("reset"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runReset(app)
		},
	}
}

func runReset(app *App) error {
	runtimeDir, err := app.RuntimeDir()
	if err != nil {
		return err
	}

	// Refuse to run if the loop is currently running.
	lockPath := filepath.Join(runtimeDir, "noodle.lock")
	lock, err := lockfile.TryLock(lockPath)
	if err != nil {
		var locked *lockfile.AlreadyLockedError
		if errors.As(err, &locked) {
			return fmt.Errorf("noodle is running (PID %d) — stop it before resetting", locked.HolderPID)
		}
		// Lock file doesn't exist or other benign error — proceed.
		if !os.IsNotExist(err) {
			return err
		}
	}
	if lock != nil {
		lock.Close()
	}

	files := []string{
		"orders.json",
		"orders-next.json",
		"mise.json",
		"failed.json",
		"pending-review.json",
		"status.json",
		"tickets.json",
		"control.ndjson",
		"control-ack.ndjson",
		"control.lock",
		"last-applied-seq",
		"loop-events.ndjson",
		"noodle.lock",
	}

	dirs := []string{
		"sessions",
		"quality",
	}

	for _, name := range files {
		path := filepath.Join(runtimeDir, name)
		if err := os.Remove(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("remove %s: %w", name, err)
		}
		fmt.Println("removed", name)
	}

	for _, name := range dirs {
		path := filepath.Join(runtimeDir, name)
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("stat %s: %w", name, err)
		}
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove %s: %w", name, err)
		}
		fmt.Println("removed", name+"/")
	}

	fmt.Println("runtime state reset")
	return nil
}

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
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if lock != nil {
		lock.Close()
	}

	if err := os.RemoveAll(runtimeDir); err != nil {
		return fmt.Errorf("remove %s: %w", runtimeDir, err)
	}
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return fmt.Errorf("recreate %s: %w", runtimeDir, err)
	}

	fmt.Println("runtime state reset")
	return nil
}

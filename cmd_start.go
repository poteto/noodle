package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "charm.land/bubbletea/v2"
	"github.com/poteto/noodle/cmdmeta"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/loop"
	"github.com/poteto/noodle/tui"
	"github.com/spf13/cobra"
)

type startRuntimeLoop interface {
	Cycle(ctx context.Context) error
	Run(ctx context.Context) error
	Shutdown()
}

var newStartRuntimeLoop = func(projectDir, noodleBin string, cfg config.Config) startRuntimeLoop {
	return loop.New(projectDir, noodleBin, cfg, loop.Dependencies{})
}

var runStartTUI = runTUI

type startOptions struct {
	once     bool
	headless bool
}

func newStartCmd(app *App) *cobra.Command {
	var opts startOptions
	cmd := &cobra.Command{
		Use:   "start",
		Short: cmdmeta.Short("start"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runStart(cmd.Context(), app, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.once, "once", false, "Run one scheduling cycle and exit")
	cmd.Flags().BoolVar(&opts.headless, "headless", false, "Run without the TUI")
	return cmd
}

func runStart(ctx context.Context, app *App, opts startOptions) error {
	cwd, err := app.ProjectDir()
	if err != nil {
		return err
	}
	noodleBin, err := app.NoodleBinaryPath()
	if err != nil {
		return err
	}
	runtimeDir, err := app.RuntimeDir()
	if err != nil {
		return err
	}

	runtimeLoop := newStartRuntimeLoop(cwd, noodleBin, app.Config)
	if opts.once {
		return runtimeLoop.Cycle(ctx)
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()
	defer runtimeLoop.Shutdown()
	if !opts.headless && isInteractiveTerminal() {
		return runStartWithTUI(ctx, cancel, runtimeLoop, runtimeDir)
	}
	return runtimeLoop.Run(ctx)
}

func runTUI(ctx context.Context, runtimeDir string) error {
	model := tui.NewModel(tui.Options{
		RuntimeDir: runtimeDir,
	})
	program := tea.NewProgram(model, tea.WithContext(ctx))
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
	return nil
}

func runStartWithTUI(
	ctx context.Context,
	cancel context.CancelFunc,
	runtimeLoop startRuntimeLoop,
	runtimeDir string,
) error {
	loopErrCh := make(chan error, 1)
	tuiErrCh := make(chan error, 1)
	go func() {
		loopErrCh <- runtimeLoop.Run(ctx)
	}()
	go func() {
		tuiErrCh <- runStartTUI(ctx, runtimeDir)
	}()

	var loopErr error
	var tuiErr error
	loopDone := false
	tuiDone := false
	for !loopDone || !tuiDone {
		select {
		case err := <-loopErrCh:
			loopDone = true
			if err != nil && !errors.Is(err, context.Canceled) {
				loopErr = err
			}
			cancel()
		case err := <-tuiErrCh:
			tuiDone = true
			if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, tea.ErrProgramKilled) {
				tuiErr = err
			}
			cancel()
		}
	}

	if loopErr != nil {
		return loopErr
	}
	if tuiErr != nil {
		return tuiErr
	}
	return nil
}

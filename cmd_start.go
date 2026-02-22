package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/loop"
)

type startRuntimeLoop interface {
	Cycle(ctx context.Context) error
	Run(ctx context.Context) error
}

var newStartRuntimeLoop = func(projectDir, noodleBin string, cfg config.Config) startRuntimeLoop {
	return loop.New(projectDir, noodleBin, cfg, loop.Dependencies{})
}

var runStartTUI = runTUI

func runStartCommand(ctx context.Context, app *App, _ []Command, args []string) error {
	flags := flag.NewFlagSet("start", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	once := flags.Bool("once", false, "Run one scheduling cycle and exit")
	if err := flags.Parse(args); err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current directory: %w", err)
	}
	noodleBin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	runtimeDir := filepath.Join(cwd, ".noodle")

	runtimeLoop := newStartRuntimeLoop(cwd, noodleBin, app.Config)
	if *once {
		return runtimeLoop.Cycle(ctx)
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if isInteractiveTerminal() {
		return runStartWithTUI(ctx, cancel, runtimeLoop, runtimeDir)
	}
	return runtimeLoop.Run(ctx)
}

func runStartWithTUI(
	ctx context.Context,
	cancel context.CancelFunc,
	runtimeLoop startRuntimeLoop,
	runtimeDir string,
) error {
	loopErrCh := make(chan error, 1)
	go func() {
		loopErrCh <- runtimeLoop.Run(ctx)
	}()

	tuiErr := runStartTUI(runtimeDir)
	cancel()
	loopErr := <-loopErrCh

	if tuiErr != nil {
		return tuiErr
	}
	if loopErr != nil && !errors.Is(loopErr, context.Canceled) {
		return loopErr
	}
	return nil
}

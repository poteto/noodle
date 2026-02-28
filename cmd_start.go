package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/poteto/noodle/cmdmeta"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/lockfile"
	"github.com/poteto/noodle/loop"
	"github.com/poteto/noodle/server"
	"github.com/spf13/cobra"
)

type startRuntimeLoop interface {
	Cycle(ctx context.Context) error
	Run(ctx context.Context) error
	Shutdown()
	State() loop.LoopState
}

var newStartRuntimeLoop = func(projectDir, noodleBin string, cfg config.Config) startRuntimeLoop {
	return loop.New(projectDir, noodleBin, cfg, loop.Dependencies{})
}

var openBrowserFunc = openBrowser

type startOptions struct {
	once bool
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

	lock, err := lockfile.TryLock(filepath.Join(runtimeDir, "noodle.lock"))
	if err != nil {
		var locked *lockfile.AlreadyLockedError
		if errors.As(err, &locked) {
			return fmt.Errorf("another noodle instance is running on this project (PID %d)", locked.HolderPID)
		}
		return err
	}
	defer lock.Close()

	runtimeLoop := newStartRuntimeLoop(cwd, noodleBin, app.Config)
	if opts.once {
		return runtimeLoop.Cycle(ctx)
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()
	defer runtimeLoop.Shutdown()

	var warnings []string
	for _, w := range app.Validation.Warnings() {
		warnings = append(warnings, w.Message)
	}

	interactive := isInteractiveTerminal()
	if shouldStartServer(app.Config.Server, interactive) {
		go func() { _ = runWebServer(ctx, runtimeDir, app.Config, runtimeLoop, warnings) }()
	}
	return runtimeLoop.Run(ctx)
}

// shouldStartServer determines if the web server should start.
// If Enabled is explicitly set, use that. Otherwise auto-enable for interactive
// sessions or when NOODLE_SERVER=1 (used by pnpm dev).
func shouldStartServer(cfg config.ServerConfig, interactive bool) bool {
	if cfg.Enabled != nil {
		return *cfg.Enabled
	}
	if os.Getenv("NOODLE_SERVER") == "1" {
		return true
	}
	return interactive
}

func runWebServer(ctx context.Context, runtimeDir string, cfg config.Config, provider server.LoopStateProvider, warnings []string) error {
	port := cfg.Server.Port
	if port == 0 {
		port = 3000
	}
	addr, err := server.FindPort(port)
	if err != nil {
		return err
	}
	srv := server.New(server.Options{
		RuntimeDir:        runtimeDir,
		Addr:              addr,
		UI:                uiClientFS(),
		Config:            &cfg,
		LoopStateProvider: provider,
		Warnings:          warnings,
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	srv.WaitReady()
	if os.Getenv("NOODLE_NO_BROWSER") != "1" {
		url := "http://" + srv.Addr()
		if devURL := os.Getenv("NOODLE_BROWSER_URL"); devURL != "" {
			url = devURL
		}
		openBrowserFunc(url)
	}

	return <-errCh
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	// Best-effort; ignore errors (e.g. no display, no browser).
	_ = cmd.Start()
}

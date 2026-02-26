package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	tea "charm.land/bubbletea/v2"
	"github.com/poteto/noodle/cmdmeta"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/loop"
	"github.com/poteto/noodle/server"
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
var openBrowserFunc = openBrowser

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

	interactive := !opts.headless && isInteractiveTerminal()
	serverEnabled := shouldStartServer(app.Config.Server, interactive)

	if interactive {
		return runStartWithTUI(ctx, cancel, runtimeLoop, runtimeDir, app.Config, serverEnabled)
	}
	if serverEnabled {
		go func() { _ = runWebServer(ctx, runtimeDir, app.Config) }()
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
	cfg config.Config,
	serverEnabled bool,
) error {
	loopErrCh := make(chan error, 1)
	tuiErrCh := make(chan error, 1)
	serverErrCh := make(chan error, 1)
	go func() {
		loopErrCh <- runtimeLoop.Run(ctx)
	}()
	go func() {
		tuiErrCh <- runStartTUI(ctx, runtimeDir)
	}()

	serverDone := true // assume done unless we actually start it
	if serverEnabled {
		serverDone = false
		go func() {
			serverErrCh <- runWebServer(ctx, runtimeDir, cfg)
		}()
	}

	var loopErr error
	var tuiErr error
	loopDone := false
	tuiDone := false
	for !loopDone || !tuiDone || !serverDone {
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
		case err := <-serverErrCh:
			serverDone = true
			if err != nil && !errors.Is(err, context.Canceled) {
				// Server errors are non-fatal; log but don't propagate.
				fmt.Fprintf(os.Stderr, "web server: %v\n", err)
			}
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

func runWebServer(ctx context.Context, runtimeDir string, cfg config.Config) error {
	port := cfg.Server.Port
	if port == 0 {
		port = 3000
	}
	addr, err := server.FindPort(port)
	if err != nil {
		return err
	}
	srv := server.New(server.Options{
		RuntimeDir: runtimeDir,
		Addr:       addr,
		UI:         uiClientFS(),
		Config:     &cfg,
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

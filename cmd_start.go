package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	charmlog "github.com/charmbracelet/log"
	"github.com/poteto/noodle/cmdmeta"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/lockfile"
	"github.com/poteto/noodle/internal/statever"
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

var newStartRuntimeLoop = func(projectDir, noodleBin string, cfg config.Config, deps loop.Dependencies) startRuntimeLoop {
	return loop.New(projectDir, noodleBin, cfg, deps)
}

var openBrowserFunc = openBrowser
var launchBrowserCommandFunc = launchBrowserCommand

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

	stateMarkerPath := filepath.Join(runtimeDir, "state.json")
	if err := statever.CheckCompatibility(stateMarkerPath); err != nil {
		return fmt.Errorf("state compatibility check failed at %s: %w", stateMarkerPath, err)
	}

	broker := server.NewSessionEventBroker()
	apiLogger := newAPILogger(os.Stderr)

	runtimeLoop := newStartRuntimeLoop(cwd, noodleBin, app.Config, loop.Dependencies{
		EventSink: broker,
		Logger:    slog.New(apiLogger),
	})
	interactive := isInteractiveTerminal()
	startServer := shouldStartServer(app.Config.Server, interactive)

	apiLogger.Info("start initialized",
		"project", cwd,
		"runtime_dir", runtimeDir,
		"once", opts.once,
		"server_enabled", startServer,
	)
	if opts.once {
		apiLogger.Info("running single scheduling cycle")
		return runtimeLoop.Cycle(ctx)
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()
	defer runtimeLoop.Shutdown()

	var warnings []string
	for _, w := range app.Validation.Warnings() {
		warnings = append(warnings, w.Message)
	}

	if startServer {
		go func() {
			if err := runWebServer(ctx, runtimeDir, app.Config, runtimeLoop, warnings, broker, apiLogger.With("component", "server")); err != nil {
				apiLogger.Error("web server exited", "error", err)
			}
		}()
	} else {
		apiLogger.Info("web server disabled", "interactive", interactive)
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

func runWebServer(
	ctx context.Context,
	runtimeDir string,
	cfg config.Config,
	provider server.LoopStateProvider,
	warnings []string,
	broker *server.SessionEventBroker,
	logger *charmlog.Logger,
) error {
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
		Broker:            broker,
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	srv.WaitReady()
	displayAddr := normalizeLoopbackHost(srv.Addr())
	logger.Info("listening", "addr", displayAddr)

	url := "http://" + displayAddr
	if devURL := os.Getenv("NOODLE_BROWSER_URL"); devURL != "" {
		url = devURL
	}
	if os.Getenv("NOODLE_NO_BROWSER") != "1" {
		logger.Info("opening browser", "url", url)
		_ = openBrowserFunc(url)
	} else {
		logger.Info("browser launch skipped", "reason", "NOODLE_NO_BROWSER=1", "url", url)
	}

	return <-errCh
}

func normalizeLoopbackHost(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		host = "localhost"
	}
	return net.JoinHostPort(host, port)
}

func newAPILogger(w io.Writer) *charmlog.Logger {
	return charmlog.NewWithOptions(w, charmlog.Options{
		Level:           charmlog.InfoLevel,
		Prefix:          "api",
		ReportTimestamp: true,
		TimeFormat:      time.RFC3339Nano,
	})
}

func openBrowser(url string) StartFailureEnvelope {
	err := launchBrowserCommandFunc(url)
	return newStartWarningOnlyEnvelope("browser launch handled as best-effort", err)
}

func launchBrowserCommand(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

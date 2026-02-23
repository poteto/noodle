package monitor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/internal/shellx"
)

type commandRunner func(name string, args ...string) error

type TmuxObserver struct {
	runtimeDir string
	run        commandRunner
}

func NewTmuxObserver(runtimeDir string) *TmuxObserver {
	return &TmuxObserver{
		runtimeDir: strings.TrimSpace(runtimeDir),
		run: func(name string, args ...string) error {
			return exec.Command(name, args...).Run()
		},
	}
}

func (o *TmuxObserver) Observe(sessionID string) (Observation, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return Observation{}, fmt.Errorf("session ID is required")
	}
	if o.runtimeDir == "" {
		return Observation{}, fmt.Errorf("runtime directory is required")
	}

	canonicalPath := filepath.Join(o.runtimeDir, "sessions", sessionID, "canonical.ndjson")
	stat, err := os.Stat(canonicalPath)
	if err != nil && !os.IsNotExist(err) {
		return Observation{}, fmt.Errorf("stat canonical events: %w", err)
	}

	obs := Observation{SessionID: sessionID}
	if stat != nil {
		obs.LogMTime = stat.ModTime().UTC()
		obs.LogSize = stat.Size()
	}

	tmuxName := monitorTmuxName(sessionID)
	if err := o.run("tmux", "has-session", "-t", tmuxName); err == nil {
		obs.Alive = true
	}
	return obs, nil
}

func monitorTmuxName(sessionID string) string {
	return "noodle-" + shellx.SanitizeToken(sessionID, "cook")
}

package monitor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	return "noodle-" + sanitizeToken(sessionID, "cook")
}

func sanitizeToken(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = fallback
	}

	var out strings.Builder
	lastHyphen := false
	for _, r := range value {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			out.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			out.WriteByte('-')
			lastHyphen = true
		}
	}

	result := strings.Trim(out.String(), "-")
	if result == "" {
		result = fallback
	}
	if len(result) > 48 {
		result = strings.Trim(result[:48], "-")
	}
	if result == "" {
		return fallback
	}
	return result
}

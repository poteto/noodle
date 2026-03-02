//go:build windows

package dispatcher

import (
	"errors"
	"os"
	"os/exec"
)

func configureChildProcess(cmd *exec.Cmd) {
	// No process-group configuration required for Windows builds.
}

func terminateProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	err := cmd.Process.Signal(os.Interrupt)
	if err == nil || errors.Is(err, os.ErrProcessDone) {
		return nil
	}
	return cmd.Process.Kill()
}

func forceKillProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	err := cmd.Process.Kill()
	if errors.Is(err, os.ErrProcessDone) {
		return nil
	}
	return err
}

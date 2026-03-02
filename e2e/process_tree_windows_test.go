//go:build e2e && windows

package e2e

import "os/exec"

func configureProcessGroup(_ *exec.Cmd) {}

func killCommandProcessTree(_ *exec.Cmd) {}

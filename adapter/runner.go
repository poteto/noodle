package adapter

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/shellx"
)

type RunOptions struct {
	Args  []string
	Stdin []byte
}

type Runner struct {
	projectDir string
	config     config.Config
}

func NewRunner(projectDir string, cfg config.Config) *Runner {
	return &Runner{
		projectDir: strings.TrimSpace(projectDir),
		config:     cfg,
	}
}

func (r *Runner) Run(ctx context.Context, adapterName, action string, options RunOptions) (string, error) {
	adapterName = strings.TrimSpace(adapterName)
	action = strings.TrimSpace(action)
	if adapterName == "" {
		return "", fmt.Errorf("adapter name is required")
	}
	if action == "" {
		return "", fmt.Errorf("adapter action is required")
	}

	adapterConfig, ok := r.config.Adapters[adapterName]
	if !ok {
		return "", fmt.Errorf("adapter %q is not configured", adapterName)
	}
	command := strings.TrimSpace(adapterConfig.Scripts[action])
	if command == "" {
		return "", fmt.Errorf("adapter %q action %q script is not configured", adapterName, action)
	}

	dir := r.projectDir
	if dir == "" {
		dir = "."
	}
	fullCommand := commandWithArgs(command, options.Args)
	cmd := exec.CommandContext(ctx, "sh", "-c", fullCommand)
	cmd.Dir = dir

	if len(options.Stdin) > 0 {
		cmd.Stdin = bytes.NewReader(options.Stdin)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrText := strings.TrimSpace(stderr.String())
		if stderrText == "" {
			stderrText = "no stderr"
		}
		return "", fmt.Errorf(
			"run adapter %s.%s command %q: %s: %w",
			adapterName,
			action,
			fullCommand,
			stderrText,
			err,
		)
	}

	return stdout.String(), nil
}

func (r *Runner) SyncBacklog(ctx context.Context) ([]BacklogItem, error) {
	output, err := r.Run(ctx, "backlog", "sync", RunOptions{})
	if err != nil {
		return nil, err
	}
	return ParseBacklogItems(output)
}

func commandWithArgs(command string, args []string) string {
	if len(args) == 0 {
		return command
	}

	var b strings.Builder
	b.WriteString(command)
	for _, arg := range args {
		b.WriteByte(' ')
		b.WriteString(shellx.Quote(arg))
	}
	return b.String()
}


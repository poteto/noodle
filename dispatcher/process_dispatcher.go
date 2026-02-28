package dispatcher

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/internal/filex"
	"github.com/poteto/noodle/skill"
	wt "github.com/poteto/noodle/worktree"
)

// ProcessDispatcherConfig configures a process dispatcher.
type ProcessDispatcherConfig struct {
	ProjectDir      string
	RuntimeDir      string
	NoodleBin       string
	SkillResolver   skill.Resolver
	ProviderConfigs ProviderConfigs
	RuntimeDefault  string // command template from config, empty = built-in
	RuntimeKind     string // runtime kind this dispatcher instance services
	Sink            SessionEventSink
}

// ProcessDispatcher dispatches provider sessions as direct child processes
// with bidirectional pipes. Replaces the tmux-based dispatcher.
type ProcessDispatcher struct {
	projectDir      string
	runtimeDir      string
	noodleBin       string
	skillResolver   skill.Resolver
	providerConfigs ProviderConfigs
	runtimeDefault  string
	runtimeKind     string
	sink            SessionEventSink
}

func NewProcessDispatcher(config ProcessDispatcherConfig) *ProcessDispatcher {
	return &ProcessDispatcher{
		projectDir:      strings.TrimSpace(config.ProjectDir),
		runtimeDir:      strings.TrimSpace(config.RuntimeDir),
		noodleBin:       strings.TrimSpace(config.NoodleBin),
		skillResolver:   config.SkillResolver,
		providerConfigs: config.ProviderConfigs,
		runtimeDefault:  strings.TrimSpace(config.RuntimeDefault),
		runtimeKind:     NormalizeRuntime(config.RuntimeKind),
		sink:            config.Sink,
	}
}

func (d *ProcessDispatcher) Dispatch(ctx context.Context, req DispatchRequest) (Session, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	reqRuntime := strings.TrimSpace(req.Runtime)
	if reqRuntime == "" {
		reqRuntime = d.runtimeKind
	} else {
		reqRuntime = NormalizeRuntime(reqRuntime)
	}
	if reqRuntime != d.runtimeKind {
		return nil, fmt.Errorf("runtime %q not configured", reqRuntime)
	}
	req.Runtime = reqRuntime

	if req.AllowPrimaryCheckout {
		req.WorktreePath = strings.TrimSpace(req.WorktreePath)
		if req.WorktreePath == "" {
			req.WorktreePath = d.projectDir
		}
		if strings.TrimSpace(req.WorktreePath) == "" {
			return nil, fmt.Errorf("project directory not set")
		}
	} else {
		validWorktree, err := wt.ValidateLinkedCheckout(req.WorktreePath)
		if err != nil {
			return nil, fmt.Errorf("worktree enforcement: %w", err)
		}
		req.WorktreePath = validWorktree
	}

	if d.runtimeDir == "" {
		return nil, fmt.Errorf("runtime directory not set")
	}

	sessionID, err := generateSessionID(req.Name)
	if err != nil {
		return nil, fmt.Errorf("generate session ID: %w", err)
	}
	sessionDir, promptPath, stampedPath, canonicalPath, stderrPath := sessionPaths(d.runtimeDir, sessionID)

	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, fmt.Errorf("create session directory: %w", err)
	}
	eventWriter, err := event.NewEventWriter(d.runtimeDir, sessionID)
	if err != nil {
		return nil, fmt.Errorf("create event writer: %w", err)
	}

	skillBundle, err := resolveSkillBundle(d.skillResolver, req)
	if err != nil {
		return nil, err
	}

	preamble := buildSessionPreamble()
	systemPrompt, composedPrompt := composePrompts(req.Provider, req.Prompt, preamble, skillBundle.SystemPrompt)

	if _, err := writePromptFiles(sessionDir, promptPath, req.Prompt, composedPrompt); err != nil {
		return nil, err
	}

	cmd, err := d.buildCmd(req, systemPrompt)
	if err != nil {
		return nil, fmt.Errorf("build command: %w", err)
	}
	cmd.Dir = req.WorktreePath
	cmd.Env = append(buildDispatchEnv(req), "NOODLE_SESSION_ID="+sessionID)

	process, err := StartProcess(cmd)
	if err != nil {
		return nil, fmt.Errorf("start process: %w", err)
	}

	// For Claude, create a controller that owns stdin for live steering.
	// The prompt is sent as the first user message via stream-json.
	// For other providers, write the prompt to stdin and close.
	var controller *claudeController
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	if provider != "codex" {
		controller = newClaudeController(process.Stdin())
		// Send the composed prompt as the first user message.
		if err := controller.SendMessage(ctx, composedPrompt); err != nil {
			_ = process.Kill()
			return nil, fmt.Errorf("send initial prompt: %w", err)
		}
	} else {
		go func() {
			_, _ = io.WriteString(process.Stdin(), composedPrompt)
			_ = process.Stdin().Close()
		}()
	}

	// Drain stderr to file.
	go drainToFile(process.Stderr(), stderrPath)

	if err := WriteProcessMetadata(sessionDir, sessionID, process.PID(), nowUTC()); err != nil {
		_ = process.Kill()
		return nil, fmt.Errorf("write process metadata: %w", err)
	}
	if err := writeDispatchMetadata(d.runtimeDir, sessionID, req, nowUTC()); err != nil {
		_ = process.Kill()
		return nil, fmt.Errorf("write spawn metadata: %w", err)
	}

	session := newProcessSession(processSessionConfig{
		id:            sessionID,
		process:       process,
		eventWriter:   eventWriter,
		canonicalPath: canonicalPath,
		stampedPath:   stampedPath,
		prompt:        req.Prompt,
		warnings:      skillBundle.Warnings,
		controller:    controller,
		sink:          d.sink,
	})
	session.start(ctx)
	return session, nil
}

// buildCmd constructs an exec.Cmd for the provider. The stamp processor runs
// in-process via processSession.processStream, so no shell pipeline is needed.
func (d *ProcessDispatcher) buildCmd(req DispatchRequest, systemPrompt string) (*exec.Cmd, error) {
	runtimeCmd := strings.TrimSpace(d.runtimeDefault)
	if runtimeCmd != "" {
		return d.buildCustomRuntimeCmd(req, runtimeCmd)
	}

	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	binary := d.resolveAgentBinary(provider)
	extraArgs := d.resolveExtraArgs(provider)

	switch provider {
	case "codex":
		args := codexBaseArgs(req)
		args = append(args, extraArgs...)
		return exec.Command(binary, args...), nil

	default: // claude
		args := claudeBaseArgs(req, systemPrompt)
		// Replace -p (pipe mode) with --input-format stream-json for
		// bidirectional streaming. The prompt is sent as a user message
		// through the claudeController rather than piped to stdin.
		args = replaceFlag(args, "-p", "--input-format", "stream-json")
		args = append(args, extraArgs...)
		return exec.Command(binary, args...), nil
	}
}

// buildCustomRuntimeCmd handles user-provided runtime command templates.
func (d *ProcessDispatcher) buildCustomRuntimeCmd(req DispatchRequest, runtimeCmd string) (*exec.Cmd, error) {
	vars := map[string]string{
		"repo":   req.WorktreePath,
		"prompt": filepath.Join(d.runtimeDir, "sessions", req.Name, "input.txt"),
		"skill":  req.Skill,
		"brief":  filepath.Join(d.runtimeDir, "mise.json"),
	}
	resolved := resolveTemplateVars(runtimeCmd, vars)
	return exec.Command("sh", "-c", resolved), nil
}

func (d *ProcessDispatcher) resolveAgentBinary(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "codex":
		if path := strings.TrimSpace(d.providerConfigs.Codex.Path); path != "" {
			candidate := filepath.Join(filex.ExpandHome(path), "codex")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		return "codex"
	default:
		if path := strings.TrimSpace(d.providerConfigs.Claude.Path); path != "" {
			candidate := filepath.Join(filex.ExpandHome(path), "claude")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		return "claude"
	}
}

func (d *ProcessDispatcher) resolveExtraArgs(provider string) []string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "codex":
		return d.providerConfigs.Codex.Args
	default:
		return d.providerConfigs.Claude.Args
	}
}

// drainToFile reads from r and writes to the named file.
func drainToFile(r io.ReadCloser, path string) {
	if r == nil || strings.TrimSpace(path) == "" {
		return
	}
	defer r.Close()
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = io.Copy(f, r)
}

// replaceFlag removes old from args and appends the replacement key-value pair.
func replaceFlag(args []string, old string, newKey, newVal string) []string {
	out := make([]string, 0, len(args)+1)
	for _, a := range args {
		if a != old {
			out = append(out, a)
		}
	}
	return append(out, newKey, newVal)
}

var _ Dispatcher = (*ProcessDispatcher)(nil)

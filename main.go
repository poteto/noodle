package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/loop"
	"github.com/poteto/noodle/skill"
	"github.com/poteto/noodle/worktree"
)

type repairLaunchResult struct {
	SessionID    string
	WorktreePath string
}

type missingScriptDiagnostic struct {
	FieldPath string
	Adapter   string
	Action    string
	Path      string
}

type repairSelectionPrompt func(input io.Reader, w io.Writer) (provider string, selected bool, err error)
type repairSessionLauncher func(
	ctx context.Context,
	app *App,
	provider string,
	prompt string,
) (repairLaunchResult, error)

var (
	repairSelectionPromptFunc repairSelectionPrompt = promptRepairSelection
	repairSessionLauncherFunc repairSessionLauncher = startRepairSession
	terminalInteractiveCheck                        = isInteractiveTerminal
)

func main() {
	if err := NewRootCmd().ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func reportConfigDiagnostics(
	ctx context.Context,
	w io.Writer,
	input io.Reader,
	commandName string,
	app *App,
	validation config.ValidationResult,
) error {
	if len(validation.Diagnostics) == 0 {
		return nil
	}

	missingScripts, passthrough := splitMissingScriptDiagnostics(validation.Diagnostics)
	for _, diagnostic := range passthrough {
		line := fmt.Sprintf(
			"config %s: %s: %s",
			strings.ToLower(string(diagnostic.Severity)),
			diagnostic.FieldPath,
			diagnostic.Message,
		)
		if diagnostic.Fix != "" {
			line += " Fix: " + diagnostic.Fix
		}
		fmt.Fprintln(w, line)
	}

	if len(missingScripts) > 0 {
		writeMissingScriptDiagnostics(w, missingScripts)
		repairPrompt := buildRepairPrompt(missingScripts)
		writeRepairPromptBlock(w, repairPrompt)

		if commandName == "start" && len(validation.Fatals()) == 0 && terminalInteractiveCheck() {
			provider, selected, err := repairSelectionPromptFunc(input, w)
			if err != nil {
				return fmt.Errorf("read repair selection: %w", err)
			}
			if selected {
				result, err := repairSessionLauncherFunc(ctx, app, provider, repairPrompt)
				if err != nil {
					return fmt.Errorf("start repair session: %w", err)
				}
				fmt.Fprintf(
					w,
					"config repair: started %s session %s in %s\n",
					provider,
					result.SessionID,
					result.WorktreePath,
				)
				return fmt.Errorf("repair session started; rerun `noodle start` after repair completes")
			}
		}
	}

	if commandName == "start" && len(validation.Fatals()) > 0 {
		return fmt.Errorf("fatal config diagnostics prevent start")
	}

	return nil
}

func splitMissingScriptDiagnostics(
	diagnostics []config.ConfigDiagnostic,
) ([]missingScriptDiagnostic, []config.ConfigDiagnostic) {
	missing := make([]missingScriptDiagnostic, 0)
	passthrough := make([]config.ConfigDiagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		parsed, ok := parseMissingScriptDiagnostic(diagnostic)
		if !ok {
			passthrough = append(passthrough, diagnostic)
			continue
		}
		missing = append(missing, parsed)
	}
	sort.Slice(missing, func(i, j int) bool {
		if missing[i].Adapter == missing[j].Adapter {
			return missing[i].Action < missing[j].Action
		}
		return missing[i].Adapter < missing[j].Adapter
	})
	return missing, passthrough
}

func parseMissingScriptDiagnostic(
	diagnostic config.ConfigDiagnostic,
) (missingScriptDiagnostic, bool) {
	if diagnostic.Severity != config.DiagnosticSeverityRepairable ||
		diagnostic.Code != config.DiagnosticCodeAdapterScriptMissing {
		return missingScriptDiagnostic{}, false
	}
	adapter := strings.TrimSpace(diagnostic.Meta["adapter"])
	action := strings.TrimSpace(diagnostic.Meta["action"])
	path := strings.TrimSpace(diagnostic.Meta["path"])
	if adapter == "" || action == "" || path == "" {
		return missingScriptDiagnostic{}, false
	}
	return missingScriptDiagnostic{
		FieldPath: diagnostic.FieldPath,
		Adapter:   adapter,
		Action:    action,
		Path:      path,
	}, true
}

func writeMissingScriptDiagnostics(w io.Writer, missing []missingScriptDiagnostic) {
	fmt.Fprintf(
		w,
		"config repairable: %d adapter script path(s) are missing.\n",
		len(missing),
	)
	byAdapter := map[string][]missingScriptDiagnostic{}
	adapterNames := make([]string, 0)
	for _, item := range missing {
		if _, exists := byAdapter[item.Adapter]; !exists {
			adapterNames = append(adapterNames, item.Adapter)
		}
		byAdapter[item.Adapter] = append(byAdapter[item.Adapter], item)
	}
	sort.Strings(adapterNames)
	for _, adapter := range adapterNames {
		fmt.Fprintf(w, "  %s:\n", adapter)
		items := byAdapter[adapter]
		sort.Slice(items, func(i, j int) bool { return items[i].Action < items[j].Action })
		for _, item := range items {
			fmt.Fprintf(
				w,
				"    - %s -> %s (%s)\n",
				item.Action,
				item.Path,
				item.FieldPath,
			)
		}
	}
}

func writeRepairPromptBlock(w io.Writer, prompt string) {
	fmt.Fprintln(w, "config repair prompt:")
	fmt.Fprintln(w, "-----")
	fmt.Fprintln(w, prompt)
	fmt.Fprintln(w, "-----")
}

func buildRepairPrompt(missing []missingScriptDiagnostic) string {
	var builder strings.Builder
	builder.WriteString("Repair Noodle adapter script configuration so `noodle start` no longer reports missing script paths.\n")
	builder.WriteString("Fix each missing entry by either creating an executable script at that path or updating noodle.toml to a valid executable path.\n")
	builder.WriteString("Keep the fix minimal and avoid broad refactors.\n\n")
	builder.WriteString("Missing adapter scripts:\n")
	for _, item := range missing {
		fmt.Fprintf(
			&builder,
			"- %s (%s): %s\n",
			item.FieldPath,
			item.Adapter,
			item.Path,
		)
	}
	builder.WriteString("\nRequired verification:\n")
	builder.WriteString("- Re-run `noodle start` and confirm these repairable diagnostics are gone.\n")
	builder.WriteString("- If Go files changed, run `go test ./...`.\n")
	builder.WriteString("\nReturn a short summary of files changed and the verification results.")
	return builder.String()
}

func promptRepairSelection(input io.Reader, w io.Writer) (string, bool, error) {
	if input == nil {
		return "", false, nil
	}
	reader := bufio.NewReader(input)
	for attempts := 0; attempts < 3; attempts++ {
		fmt.Fprintln(w, "Run automatic repair now?")
		fmt.Fprintln(w, "  1) Claude")
		fmt.Fprintln(w, "  2) Codex")
		fmt.Fprintln(w, "  3) Skip")
		fmt.Fprint(w, "Select [1-3]: ")

		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", false, err
		}
		choice := strings.ToLower(strings.TrimSpace(line))
		switch choice {
		case "1", "claude", "c":
			return "claude", true, nil
		case "2", "codex", "x":
			return "codex", true, nil
		case "", "3", "skip", "s":
			return "", false, nil
		default:
			fmt.Fprintf(w, "Invalid selection %q. Please choose 1, 2, or 3.\n", choice)
		}
		if err == io.EOF {
			return "", false, nil
		}
	}
	return "", false, nil
}

func startRepairSession(
	ctx context.Context,
	app *App,
	provider string,
	prompt string,
) (repairLaunchResult, error) {
	if app == nil {
		return repairLaunchResult{}, fmt.Errorf("application context not available")
	}
	if !app.Validation.CanSpawn() {
		return repairLaunchResult{}, fmt.Errorf("fatal config diagnostics prevent spawn")
	}

	worktreeApp, err := worktree.NewApp()
	if err != nil {
		return repairLaunchResult{}, fmt.Errorf("load worktree app: %w", err)
	}
	name := "repair-config-" + time.Now().UTC().Format("20060102-150405")
	if err := worktreeApp.Create(name); err != nil {
		return repairLaunchResult{}, fmt.Errorf("create repair worktree: %w", err)
	}
	worktreePath := filepath.Join(worktreeApp.Root, ".worktrees", name)

	noodleBin, err := os.Executable()
	if err != nil {
		_ = worktreeApp.Cleanup(name, true)
		return repairLaunchResult{}, fmt.Errorf("resolve executable path: %w", err)
	}
	runtimeDir := filepath.Join(worktreeApp.Root, ".noodle")
	resolver := skill.Resolver{SearchPaths: app.Config.Skills.Paths}
	agentDirs := dispatcher.AgentDirs{
		ClaudeDir: app.Config.Agents.ClaudeDir,
		CodexDir:  app.Config.Agents.CodexDir,
	}
	d := dispatcher.NewTmuxDispatcher(dispatcher.TmuxDispatcherConfig{
		ProjectDir:    worktreeApp.Root,
		RuntimeDir:    runtimeDir,
		NoodleBin:     noodleBin,
		SkillResolver: resolver,
		AgentDirs:     agentDirs,
	})
	request := dispatcher.DispatchRequest{
		Name:         name,
		Prompt:       prompt,
		Provider:     provider,
		Model:        repairModelForProvider(provider, app.Config),
		Skill:        loop.RepairTaskSkill(),
		WorktreePath: worktreePath,
	}
	session, err := d.Dispatch(ctx, request)
	if err != nil {
		_ = worktreeApp.Cleanup(name, true)
		return repairLaunchResult{}, err
	}
	return repairLaunchResult{
		SessionID:    session.ID(),
		WorktreePath: worktreePath,
	}, nil
}

func repairModelForProvider(provider string, loaded config.Config) string {
	if strings.EqualFold(provider, loaded.Routing.Defaults.Provider) &&
		strings.TrimSpace(loaded.Routing.Defaults.Model) != "" {
		return loaded.Routing.Defaults.Model
	}
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "codex":
		return "gpt-5.3-codex"
	default:
		return "claude-sonnet-4-6"
	}
}

func isInteractiveTerminal() bool {
	return fileLooksInteractive(os.Stdin) && fileLooksInteractive(os.Stdout)
}

func fileLooksInteractive(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

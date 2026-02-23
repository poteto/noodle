package loop

import (
	"path/filepath"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/monitor"
	"github.com/poteto/noodle/skill"
	"github.com/poteto/noodle/worktree"
)

type noOpWorktree struct{}

func (noOpWorktree) Create(string) error        { return nil }
func (noOpWorktree) Merge(string) error         { return nil }
func (noOpWorktree) Cleanup(string, bool) error { return nil }

func defaultDependencies(projectDir, runtimeDir, noodleBin string, cfg config.Config) Dependencies {
	resolver := skill.Resolver{SearchPaths: cfg.Skills.Paths}
	sp := dispatcher.NewTmuxDispatcher(dispatcher.TmuxDispatcherConfig{
		ProjectDir:     projectDir,
		RuntimeDir:     runtimeDir,
		NoodleBin:      noodleBin,
		SkillResolver:  resolver,
		RuntimeDefault: cfg.Runtime.Default,
		AgentDirs: dispatcher.AgentDirs{
			ClaudeDir: cfg.Agents.ClaudeDir,
			CodexDir:  cfg.Agents.CodexDir,
		},
	})
	wtApp, _ := worktree.NewApp()
	var wt WorktreeManager
	if wtApp != nil {
		wtApp.CmdPrefix = "noodle worktree"
		wtApp.Quiet = true
		wt = wtApp
	} else {
		wt = noOpWorktree{}
	}
	return Dependencies{
		Dispatcher: sp,
		Worktree:  wt,
		Adapter:   adapter.NewRunner(projectDir, cfg),
		Mise:      mise.NewBuilder(projectDir, cfg),
		Monitor:   monitor.NewMonitor(runtimeDir),
		Now:       time.Now,
		QueueFile: filepath.Join(runtimeDir, "queue.json"),
	}
}

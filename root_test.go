package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommandRegistersAllSubcommands(t *testing.T) {
	root := NewRootCmd()

	expected := []string{
		"start", "skills", "schema", "status", "debug",
		"worktree", "stamp", "dispatch", "mise", "plan",
	}

	found := map[string]bool{}
	for _, cmd := range root.Commands() {
		found[cmd.Name()] = true
	}

	for _, name := range expected {
		if !found[name] {
			t.Fatalf("%q command not registered", name)
		}
	}
}

func TestWorktreeCommandRegistersSubcommands(t *testing.T) {
	root := NewRootCmd()
	var wt *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Name() == "worktree" {
			wt = cmd
			break
		}
	}
	if wt == nil {
		t.Fatal("worktree command not registered")
	}

	expected := map[string]bool{
		"hook": true, "create": true, "merge": true,
		"cleanup": true, "list": true, "prune": true,
	}
	for _, cmd := range wt.Commands() {
		delete(expected, cmd.Name())
	}
	for name := range expected {
		t.Fatalf("worktree %q subcommand not registered", name)
	}
}

func TestPlanCommandRegistersSubcommands(t *testing.T) {
	root := NewRootCmd()
	var pl *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Name() == "plan" {
			pl = cmd
			break
		}
	}
	if pl == nil {
		t.Fatal("plan command not registered")
	}

	expected := map[string]bool{
		"create": true, "done": true, "phase-add": true, "list": true,
	}
	for _, cmd := range pl.Commands() {
		delete(expected, cmd.Name())
	}
	for name := range expected {
		t.Fatalf("plan %q subcommand not registered", name)
	}
}

func TestSkillsCommandRegistersSubcommands(t *testing.T) {
	root := NewRootCmd()
	var sk *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Name() == "skills" {
			sk = cmd
			break
		}
	}
	if sk == nil {
		t.Fatal("skills command not registered")
	}

	found := false
	for _, cmd := range sk.Commands() {
		if cmd.Name() == "list" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("skills list subcommand not registered")
	}
}

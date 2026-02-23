package main

import (
	"context"
	"io"
	"testing"
)

func TestWorktreeCreateDispatches(t *testing.T) {
	originalFactory := newWorktreeCommandApp
	t.Cleanup(func() {
		newWorktreeCommandApp = originalFactory
	})

	fake := &fakeWorktreeCommandApp{}
	newWorktreeCommandApp = func() (worktreeCommandApp, error) {
		return fake, nil
	}

	cmd := newWorktreeCmd(nil)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"create", "feat-a"})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("worktree create: %v", err)
	}
	if fake.createName != "feat-a" {
		t.Fatalf("create name = %q", fake.createName)
	}
}

func TestWorktreeCleanupWithForce(t *testing.T) {
	originalFactory := newWorktreeCommandApp
	t.Cleanup(func() {
		newWorktreeCommandApp = originalFactory
	})

	fake := &fakeWorktreeCommandApp{}
	newWorktreeCommandApp = func() (worktreeCommandApp, error) {
		return fake, nil
	}

	cmd := newWorktreeCmd(nil)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"cleanup", "--force", "feat-a"})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("worktree cleanup: %v", err)
	}
	if fake.cleanupName != "feat-a" {
		t.Fatalf("cleanup name = %q", fake.cleanupName)
	}
	if !fake.cleanupForce {
		t.Fatal("expected cleanup force=true")
	}
}

func TestWorktreeHookBypassesAppFactory(t *testing.T) {
	originalFactory := newWorktreeCommandApp
	originalHook := runWorktreeHook
	t.Cleanup(func() {
		newWorktreeCommandApp = originalFactory
		runWorktreeHook = originalHook
	})

	factoryCalled := false
	newWorktreeCommandApp = func() (worktreeCommandApp, error) {
		factoryCalled = true
		return &fakeWorktreeCommandApp{}, nil
	}

	hookCalled := false
	runWorktreeHook = func(_ io.Reader, _ io.Writer) error {
		hookCalled = true
		return nil
	}

	cmd := newWorktreeCmd(nil)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"hook"})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("worktree hook: %v", err)
	}
	if factoryCalled {
		t.Fatal("expected hook to bypass worktree app factory")
	}
	if !hookCalled {
		t.Fatal("expected hook handler call")
	}
}

type fakeWorktreeCommandApp struct {
	createName   string
	execName     string
	execArgs     []string
	mergeName    string
	cleanupName  string
	cleanupForce bool
	listCalled   bool
	pruneCalled  bool
}

func (f *fakeWorktreeCommandApp) Create(name string) error {
	f.createName = name
	return nil
}

func (f *fakeWorktreeCommandApp) Exec(name string, args []string) error {
	f.execName = name
	f.execArgs = args
	return nil
}

func (f *fakeWorktreeCommandApp) Merge(name string) error {
	f.mergeName = name
	return nil
}

func (f *fakeWorktreeCommandApp) Cleanup(name string, force bool) error {
	f.cleanupName = name
	f.cleanupForce = force
	return nil
}

func (f *fakeWorktreeCommandApp) List() error {
	f.listCalled = true
	return nil
}

func (f *fakeWorktreeCommandApp) Prune() error {
	f.pruneCalled = true
	return nil
}

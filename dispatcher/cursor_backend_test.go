package dispatcher

import (
	"context"
	"strings"
	"testing"
)

func TestCursorBackendMethodsReturnNotImplemented(t *testing.T) {
	t.Parallel()

	backend := &CursorBackend{}
	ctx := context.Background()

	t.Run("Launch", func(t *testing.T) {
		t.Parallel()
		_, err := backend.Launch(ctx, PollLaunchConfig{})
		assertNotImplementedError(t, err)
	})

	t.Run("PollStatus", func(t *testing.T) {
		t.Parallel()
		_, err := backend.PollStatus(ctx, "remote-id")
		assertNotImplementedError(t, err)
	})

	t.Run("GetConversation", func(t *testing.T) {
		t.Parallel()
		_, err := backend.GetConversation(ctx, "remote-id")
		assertNotImplementedError(t, err)
	})

	t.Run("Stop", func(t *testing.T) {
		t.Parallel()
		err := backend.Stop(ctx, "remote-id")
		assertNotImplementedError(t, err)
	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()
		err := backend.Delete(ctx, "remote-id")
		assertNotImplementedError(t, err)
	})
}

func assertNotImplementedError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("expected not implemented error, got %v", err)
	}
}

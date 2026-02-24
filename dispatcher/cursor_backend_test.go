package dispatcher

import (
	"context"
	"strings"
	"testing"
)

func TestCursorBackendStubMethodsReturnNotImplemented(t *testing.T) {
	backend := &CursorBackend{}
	ctx := context.Background()

	assertNotImplemented := func(name string, err error) {
		t.Helper()
		if err == nil {
			t.Fatalf("%s: expected error", name)
		}
		if !strings.Contains(strings.ToLower(err.Error()), "not implemented") {
			t.Fatalf("%s: error = %q, want contains not implemented", name, err.Error())
		}
	}

	_, err := backend.Launch(ctx, PollLaunchConfig{})
	assertNotImplemented("launch", err)

	_, err = backend.PollStatus(ctx, "remote-id")
	assertNotImplemented("poll status", err)

	_, err = backend.GetConversation(ctx, "remote-id")
	assertNotImplemented("get conversation", err)

	err = backend.Stop(ctx, "remote-id")
	assertNotImplemented("stop", err)

	err = backend.Delete(ctx, "remote-id")
	assertNotImplemented("delete", err)
}

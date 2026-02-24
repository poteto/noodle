package dispatcher

import (
	"context"
	"fmt"
	"os"
)

// CursorBackend is a compile-time PollingBackend stub for future Cursor support.
type CursorBackend struct{}

func (b *CursorBackend) Launch(context.Context, PollLaunchConfig) (string, error) {
	return "", b.notImplemented("launch")
}

func (b *CursorBackend) PollStatus(context.Context, string) (RemoteStatus, error) {
	return RemoteStatusUnknown, b.notImplemented("poll status")
}

func (b *CursorBackend) GetConversation(context.Context, string) ([]ConversationMessage, error) {
	return nil, b.notImplemented("get conversation")
}

func (b *CursorBackend) Stop(context.Context, string) error {
	return b.notImplemented("stop")
}

func (b *CursorBackend) Delete(context.Context, string) error {
	return b.notImplemented("delete")
}

func (b *CursorBackend) notImplemented(operation string) error {
	err := fmt.Errorf("cursor backend not implemented")
	fmt.Fprintf(os.Stderr, "cursor backend %s: %v\n", operation, err)
	return err
}

var _ PollingBackend = (*CursorBackend)(nil)

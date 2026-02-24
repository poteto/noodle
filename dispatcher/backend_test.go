package dispatcher

import "context"

type streamingBackendStub struct{}

func (s *streamingBackendStub) Start(context.Context, StreamStartConfig) (StreamHandle, error) {
	return StreamHandle{}, nil
}

func (s *streamingBackendStub) IsAlive(context.Context, StreamHandle) (bool, error) {
	return true, nil
}

func (s *streamingBackendStub) Kill(context.Context, StreamHandle) error {
	return nil
}

type pollingBackendStub struct{}

func (s *pollingBackendStub) Launch(context.Context, PollLaunchConfig) (string, error) {
	return "remote-id", nil
}

func (s *pollingBackendStub) PollStatus(context.Context, string) (RemoteStatus, error) {
	return RemoteStatusRunning, nil
}

func (s *pollingBackendStub) GetConversation(context.Context, string) ([]ConversationMessage, error) {
	return nil, nil
}

func (s *pollingBackendStub) Stop(context.Context, string) error {
	return nil
}

func (s *pollingBackendStub) Delete(context.Context, string) error {
	return nil
}

var _ StreamingBackend = (*streamingBackendStub)(nil)
var _ PollingBackend = (*pollingBackendStub)(nil)

package failure

import "testing"

func TestOwnerPrefixForDisplay(t *testing.T) {
	t.Run("agent mistake includes owner prefix", func(t *testing.T) {
		got := OwnerPrefixForDisplay(FailureClassAgentMistake, FailureOwnerSchedulerAgent)
		if got != "[scheduler_agent] " {
			t.Fatalf("prefix = %q, want %q", got, "[scheduler_agent] ")
		}
	})

	t.Run("non agent mistake omits prefix", func(t *testing.T) {
		got := OwnerPrefixForDisplay(FailureClassBackendRecoverable, FailureOwnerBackend)
		if got != "" {
			t.Fatalf("prefix = %q, want empty", got)
		}
	})
}

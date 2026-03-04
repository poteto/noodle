package dispatcher

import "testing"

func TestDispatchRequestValidate(t *testing.T) {
	valid := DispatchRequest{
		Name:         "cook-a",
		Prompt:       "Say ok",
		Provider:     "claude",
		Model:        "claude-sonnet-4-6",
		WorktreePath: ".worktrees/phase-06-dispatcher",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("validate valid request: %v", err)
	}

	cases := []struct {
		name string
		req  DispatchRequest
	}{
		{
			name: "missing name",
			req: DispatchRequest{
				Prompt:       "p",
				Provider:     "claude",
				Model:        "m",
				WorktreePath: "w",
			},
		},
		{
			name: "missing prompt",
			req: DispatchRequest{
				Name:         "n",
				Provider:     "claude",
				Model:        "m",
				WorktreePath: "w",
			},
		},
		{
			name: "missing provider",
			req: DispatchRequest{
				Name:         "n",
				Prompt:       "p",
				Provider:     "",
				Model:        "m",
				WorktreePath: "w",
			},
		},
		{
			name: "missing model",
			req: DispatchRequest{
				Name:         "n",
				Prompt:       "p",
				Provider:     "claude",
				WorktreePath: "w",
			},
		},
		{
			name: "missing worktree",
			req: DispatchRequest{
				Name:     "n",
				Prompt:   "p",
				Provider: "claude",
				Model:    "m",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.req.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestDispatchRequestValidateAcceptsNonProcessProvider(t *testing.T) {
	req := DispatchRequest{
		Name:         "cook-a",
		Prompt:       "Say ok",
		Provider:     "remote-provider",
		Model:        "model-x",
		WorktreePath: ".worktrees/phase-06-dispatcher",
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("validate request: %v", err)
	}
}

func TestSessionStatusIsTerminal(t *testing.T) {
	cases := []struct {
		name   string
		status SessionStatus
		want   bool
	}{
		{name: "zero value", status: "", want: false},
		{name: "running string", status: SessionStatus("running"), want: false},
		{name: "completed", status: StatusCompleted, want: true},
		{name: "failed", status: StatusFailed, want: true},
		{name: "cancelled", status: StatusCancelled, want: true},
		{name: "killed", status: StatusKilled, want: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.status.IsTerminal(); got != tc.want {
				t.Fatalf("IsTerminal() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSessionStatusString(t *testing.T) {
	if got := StatusCompleted.String(); got != "completed" {
		t.Fatalf("String() = %q, want completed", got)
	}
}

func TestSessionOutcomeZeroValueIsNonTerminal(t *testing.T) {
	var outcome SessionOutcome
	if outcome.Status.IsTerminal() {
		t.Fatal("zero-value outcome status should be non-terminal")
	}
}

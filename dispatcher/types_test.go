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
			name: "bad provider",
			req: DispatchRequest{
				Name:         "n",
				Prompt:       "p",
				Provider:     "openai",
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

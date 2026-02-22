package spawner

import "testing"

func TestSpawnRequestValidate(t *testing.T) {
	valid := SpawnRequest{
		Name:         "cook-a",
		Prompt:       "Say ok",
		Provider:     "claude",
		Model:        "claude-sonnet-4-6",
		WorktreePath: ".worktrees/phase-06-spawner",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("validate valid request: %v", err)
	}

	cases := []struct {
		name string
		req  SpawnRequest
	}{
		{
			name: "missing name",
			req: SpawnRequest{
				Prompt:       "p",
				Provider:     "claude",
				Model:        "m",
				WorktreePath: "w",
			},
		},
		{
			name: "missing prompt",
			req: SpawnRequest{
				Name:         "n",
				Provider:     "claude",
				Model:        "m",
				WorktreePath: "w",
			},
		},
		{
			name: "bad provider",
			req: SpawnRequest{
				Name:         "n",
				Prompt:       "p",
				Provider:     "openai",
				Model:        "m",
				WorktreePath: "w",
			},
		},
		{
			name: "missing model",
			req: SpawnRequest{
				Name:         "n",
				Prompt:       "p",
				Provider:     "claude",
				WorktreePath: "w",
			},
		},
		{
			name: "missing worktree",
			req: SpawnRequest{
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

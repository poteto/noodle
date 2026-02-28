package recover

import (
	"testing"
)

func TestRecoveryChainLength(t *testing.T) {
	cases := []struct {
		name string
		want int
	}{
		{name: "task-1", want: 0},
		{name: "task-1-recover-1", want: 1},
		{name: "task-1-recover-3", want: 3},
	}
	for _, test := range cases {
		if got := RecoveryChainLength(test.name); got != test.want {
			t.Fatalf("chain length for %q = %d, want %d", test.name, got, test.want)
		}
	}
}

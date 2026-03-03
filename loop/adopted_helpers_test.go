package loop

import "testing"

func TestRecoveryChainLength(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{name: "no suffix", in: "order-1-stage-0", want: 0},
		{name: "valid suffix", in: "order-1-stage-0-recover-3", want: 3},
		{name: "negative suffix", in: "order-1-stage-0-recover--1", want: 0},
		{name: "invalid suffix", in: "order-1-stage-0-recover-x", want: 0},
		{name: "empty", in: "", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := recoveryChainLength(tt.in); got != tt.want {
				t.Fatalf("recoveryChainLength(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

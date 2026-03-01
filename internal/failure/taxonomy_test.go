package failure

import "testing"

func TestRecoverabilityForClassTable(t *testing.T) {
	tests := []struct {
		name  string
		class FailureClass
		want  FailureRecoverability
	}{
		{
			name:  "backend invariant is hard",
			class: FailureClassBackendInvariant,
			want:  FailureRecoverabilityHard,
		},
		{
			name:  "backend recoverable is recoverable",
			class: FailureClassBackendRecoverable,
			want:  FailureRecoverabilityRecoverable,
		},
		{
			name:  "agent mistake is recoverable",
			class: FailureClassAgentMistake,
			want:  FailureRecoverabilityRecoverable,
		},
		{
			name:  "agent start unrecoverable is hard",
			class: FailureClassAgentStartUnrecoverable,
			want:  FailureRecoverabilityHard,
		},
		{
			name:  "agent start retryable is recoverable",
			class: FailureClassAgentStartRetryable,
			want:  FailureRecoverabilityRecoverable,
		},
		{
			name:  "warning only is degrade",
			class: FailureClassWarningOnly,
			want:  FailureRecoverabilityDegrade,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !IsKnownClass(tt.class) {
				t.Fatalf("class %q should be known", tt.class)
			}
			got := RecoverabilityForClass(tt.class)
			if got != tt.want {
				t.Fatalf("recoverability for %q = %q, want %q", tt.class, got, tt.want)
			}
		})
	}
}

func TestRecoverabilityForClassIsDeterministic(t *testing.T) {
	for _, class := range KnownFailureClasses() {
		first := RecoverabilityForClass(class)
		for i := 0; i < 32; i++ {
			if got := RecoverabilityForClass(class); got != first {
				t.Fatalf("recoverability for %q changed at iteration %d: got %q, want %q", class, i, got, first)
			}
		}
	}
}

func TestUnknownClassUsesExplicitFallback(t *testing.T) {
	unknown := FailureClass("new_class_not_yet_mapped")
	if IsKnownClass(unknown) {
		t.Fatalf("class %q should not be known", unknown)
	}
	got := RecoverabilityForClass(unknown)
	if got != UnknownClassRecoverabilityFallback {
		t.Fatalf("recoverability for unknown class = %q, want explicit fallback %q", got, UnknownClassRecoverabilityFallback)
	}
}

func TestKnownFailureClassesReturnsCopy(t *testing.T) {
	initial := KnownFailureClasses()
	initial[0] = FailureClass("mutated")

	again := KnownFailureClasses()
	if again[0] != FailureClassBackendInvariant {
		t.Fatalf("known classes should be immutable via returned slice, got %q", again[0])
	}
}

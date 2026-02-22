package debate

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestCreateDebateUsesDefaultMaxRounds(t *testing.T) {
	store := newStore(t)

	debate, err := store.Create("Plan: auth refactor", 0)
	if err != nil {
		t.Fatalf("create debate: %v", err)
	}

	if debate.MaxRounds != DefaultMaxRounds {
		t.Fatalf("max rounds = %d, want %d", debate.MaxRounds, DefaultMaxRounds)
	}
	if debate.Target != "Plan: auth refactor" {
		t.Fatalf("target = %q", debate.Target)
	}
	if _, err := os.Stat(debate.Dir); err != nil {
		t.Fatalf("stat debate directory: %v", err)
	}
	if !regexp.MustCompile(`^plan-auth-refactor-[0-9a-f]{8}$`).MatchString(debate.ID) {
		t.Fatalf("unexpected debate ID: %q", debate.ID)
	}
}

func TestAddRoundAndReadRounds(t *testing.T) {
	store := newStore(t)
	debate, err := store.Create("Diff review", 4)
	if err != nil {
		t.Fatalf("create debate: %v", err)
	}

	round1, err := store.AddRound(debate, RoleReviewer, "Round 1 critique")
	if err != nil {
		t.Fatalf("add round 1: %v", err)
	}
	if round1.Number != 1 || round1.Role != RoleReviewer {
		t.Fatalf("round 1 = %#v", round1)
	}
	round2, err := store.AddRound(debate, RoleResponder, "Round 2 response")
	if err != nil {
		t.Fatalf("add round 2: %v", err)
	}
	if round2.Number != 2 || round2.Role != RoleResponder {
		t.Fatalf("round 2 = %#v", round2)
	}

	rounds, err := store.ReadRounds(debate)
	if err != nil {
		t.Fatalf("read rounds: %v", err)
	}
	if len(rounds) != 2 {
		t.Fatalf("round count = %d, want 2", len(rounds))
	}
	if rounds[0].Content != "Round 1 critique" {
		t.Fatalf("round 1 content = %q", rounds[0].Content)
	}
	if rounds[1].Content != "Round 2 response" {
		t.Fatalf("round 2 content = %q", rounds[1].Content)
	}

	roundPath := filepath.Join(debate.Dir, "round-1.md")
	rawRound, err := os.ReadFile(roundPath)
	if err != nil {
		t.Fatalf("read round file: %v", err)
	}
	if strings.TrimSpace(string(rawRound)) != "Round 1 critique" {
		t.Fatalf("round file content = %q", string(rawRound))
	}
}

func TestAddRoundRejectsUnexpectedRole(t *testing.T) {
	store := newStore(t)
	debate, err := store.Create("Queue item", 3)
	if err != nil {
		t.Fatalf("create debate: %v", err)
	}

	if _, err := store.AddRound(debate, RoleResponder, "wrong role"); err == nil {
		t.Fatal("expected role error")
	}
}

func TestVerdictReadWriteAndConsensus(t *testing.T) {
	store := newStore(t)
	debate, err := store.Create("Artifact", 4)
	if err != nil {
		t.Fatalf("create debate: %v", err)
	}

	if err := store.WriteVerdict(debate, Verdict{
		Consensus: true,
		Summary:   "Ready to ship",
	}); err != nil {
		t.Fatalf("write verdict true: %v", err)
	}

	verdict, err := store.ReadVerdict(debate)
	if err != nil {
		t.Fatalf("read verdict true: %v", err)
	}
	if !verdict.Consensus || verdict.Summary != "Ready to ship" {
		t.Fatalf("verdict true = %#v", verdict)
	}
	consensus, err := store.HasConsensus(debate)
	if err != nil {
		t.Fatalf("has consensus true: %v", err)
	}
	if !consensus {
		t.Fatal("consensus = false, want true")
	}

	if err := store.WriteVerdict(debate, Verdict{
		Consensus: false,
		Summary:   "Needs another round",
	}); err != nil {
		t.Fatalf("write verdict false: %v", err)
	}
	verdict, err = store.ReadVerdict(debate)
	if err != nil {
		t.Fatalf("read verdict false: %v", err)
	}
	if verdict.Consensus {
		t.Fatalf("verdict false = %#v", verdict)
	}
}

func TestMaxRoundsEnforcementAndCompletion(t *testing.T) {
	store := newStore(t)
	debate, err := store.Create("Plan item", 2)
	if err != nil {
		t.Fatalf("create debate: %v", err)
	}

	if _, err := store.AddRound(debate, RoleReviewer, "Critique"); err != nil {
		t.Fatalf("add round 1: %v", err)
	}
	complete, err := store.IsComplete(debate)
	if err != nil {
		t.Fatalf("is complete after round 1: %v", err)
	}
	if complete {
		t.Fatal("debate completed too early")
	}

	if _, err := store.AddRound(debate, RoleResponder, "Response"); err != nil {
		t.Fatalf("add round 2: %v", err)
	}
	complete, err = store.IsComplete(debate)
	if err != nil {
		t.Fatalf("is complete after round 2: %v", err)
	}
	if !complete {
		t.Fatal("debate should be complete at max rounds")
	}

	if _, err := store.AddRound(debate, RoleReviewer, "Round 3"); err == nil {
		t.Fatal("expected max rounds error")
	}
}

func TestDebateIDSlugifiesAndHashes(t *testing.T) {
	id := DebateID("  API: Add OAuth & SSO!  ")
	if !regexp.MustCompile(`^api-add-oauth-sso-[0-9a-f]{8}$`).MatchString(id) {
		t.Fatalf("unexpected debate ID: %q", id)
	}

	idRepeat := DebateID("  API: Add OAuth & SSO!  ")
	if id != idRepeat {
		t.Fatalf("debate ID changed for same target: %q vs %q", id, idRepeat)
	}

	empty := DebateID("   ")
	if !regexp.MustCompile(`^debate-[0-9a-f]{8}$`).MatchString(empty) {
		t.Fatalf("unexpected empty-target debate ID: %q", empty)
	}
}

func newStore(t *testing.T) *Store {
	t.Helper()

	store, err := NewStore(filepath.Join(t.TempDir(), "brain", "debates"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return store
}

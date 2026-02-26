package queuex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/taskreg"
	"github.com/poteto/noodle/skill"
)

func ordersTestRegistry() taskreg.Registry {
	return taskreg.NewFromSkills([]skill.SkillMeta{
		{
			Name: "execute",
			Path: "/skills/execute",
			Frontmatter: skill.Frontmatter{
				Noodle: &skill.NoodleMeta{Schedule: "When a planned item is ready"},
			},
		},
		{
			Name: "schedule",
			Path: "/skills/schedule",
			Frontmatter: skill.Frontmatter{
				Noodle: &skill.NoodleMeta{Schedule: "When queue is empty"},
			},
		},
	})
}

func TestWriteOrdersAtomicReadOrdersRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orders.json")

	now := time.Now().Truncate(time.Second)
	original := OrdersFile{
		GeneratedAt: now,
		Orders: []Order{
			{
				ID:     "order-1",
				Title:  "Implement auth",
				Status: OrderStatusActive,
				Stages: []Stage{
					{
						TaskKey:  "execute",
						Provider: "claude",
						Model:    "claude-opus-4-6",
						Status:   StageStatusPending,
					},
				},
				OnFailure: []Stage{
					{
						TaskKey:  "execute",
						Prompt:   "rollback",
						Provider: "claude",
						Model:    "claude-opus-4-6",
						Status:   StageStatusPending,
					},
				},
			},
		},
		ActionNeeded: []string{"review order-1"},
	}

	if err := WriteOrdersAtomic(path, original); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := ReadOrders(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if !got.GeneratedAt.Equal(original.GeneratedAt) {
		t.Errorf("GeneratedAt = %v, want %v", got.GeneratedAt, original.GeneratedAt)
	}
	if len(got.Orders) != 1 {
		t.Fatalf("Orders len = %d, want 1", len(got.Orders))
	}
	if got.Orders[0].ID != "order-1" {
		t.Errorf("ID = %q, want order-1", got.Orders[0].ID)
	}
	if got.Orders[0].Status != OrderStatusActive {
		t.Errorf("Status = %q, want active", got.Orders[0].Status)
	}
	if len(got.Orders[0].Stages) != 1 {
		t.Fatalf("Stages len = %d, want 1", len(got.Orders[0].Stages))
	}
	if len(got.ActionNeeded) != 1 || got.ActionNeeded[0] != "review order-1" {
		t.Errorf("ActionNeeded = %v", got.ActionNeeded)
	}
}

func TestRoundTripPreservesStageExtra(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orders.json")

	original := OrdersFile{
		Orders: []Order{
			{
				ID:     "1",
				Status: OrderStatusActive,
				Stages: []Stage{
					{
						TaskKey:  "execute",
						Provider: "claude",
						Model:    "claude-opus-4-6",
						Status:   StageStatusPending,
						Extra: map[string]json.RawMessage{
							"priority":     json.RawMessage(`42`),
							"tags":         json.RawMessage(`["urgent","backend"]`),
							"nested":       json.RawMessage(`{"key":"value"}`),
							"null_val":     json.RawMessage(`null`),
							"string_val":   json.RawMessage(`"hello"`),
						},
					},
				},
			},
		},
	}

	if err := WriteOrdersAtomic(path, original); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := ReadOrders(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	extra := got.Orders[0].Stages[0].Extra
	for key, want := range original.Orders[0].Stages[0].Extra {
		gotVal, ok := extra[key]
		if !ok {
			t.Errorf("Extra[%q] missing after round-trip", key)
			continue
		}
		if !jsonEqual(t, gotVal, want) {
			t.Errorf("Extra[%q] = %s, want %s", key, gotVal, want)
		}
	}
}

func TestRoundTripPreservesOrderOnFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orders.json")

	t.Run("with on_failure", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				{
					ID:     "1",
					Status: OrderStatusFailing,
					Stages: []Stage{
						{Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusFailed},
					},
					OnFailure: []Stage{
						{TaskKey: "execute", Prompt: "rollback", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
					},
				},
			},
		}

		if err := WriteOrdersAtomic(path, of); err != nil {
			t.Fatalf("write: %v", err)
		}
		got, err := ReadOrders(path)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if len(got.Orders[0].OnFailure) != 1 {
			t.Fatalf("OnFailure len = %d, want 1", len(got.Orders[0].OnFailure))
		}
		if got.Orders[0].OnFailure[0].Prompt != "rollback" {
			t.Errorf("OnFailure[0].Prompt = %q, want rollback", got.Orders[0].OnFailure[0].Prompt)
		}
	})

	t.Run("nil on_failure", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				{
					ID:     "2",
					Status: OrderStatusActive,
					Stages: []Stage{
						{Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
					},
					OnFailure: nil,
				},
			},
		}

		if err := WriteOrdersAtomic(path, of); err != nil {
			t.Fatalf("write: %v", err)
		}
		got, err := ReadOrders(path)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if got.Orders[0].OnFailure != nil {
			t.Errorf("OnFailure = %v, want nil", got.Orders[0].OnFailure)
		}
	})
}

func TestReadOrdersEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orders.json")

	if err := os.WriteFile(path, []byte("  \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadOrders(path)
	if err != nil {
		t.Fatalf("read empty: %v", err)
	}
	if len(got.Orders) != 0 {
		t.Fatalf("Orders len = %d, want 0", len(got.Orders))
	}
}

func TestReadOrdersMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")

	got, err := ReadOrders(path)
	if err != nil {
		t.Fatalf("read missing: %v", err)
	}
	if len(got.Orders) != 0 {
		t.Fatalf("Orders len = %d, want 0", len(got.Orders))
	}
}

func TestReadOrdersMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orders.json")

	if err := os.WriteFile(path, []byte("not json at all {{{"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadOrders(path)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "parse orders") {
		t.Fatalf("expected descriptive error, got: %v", err)
	}
}

func TestWriteOrdersAtomicUnwritableDirectory(t *testing.T) {
	path := filepath.Join("/dev/null/impossible", "orders.json")
	err := WriteOrdersAtomic(path, OrdersFile{})
	if err == nil {
		t.Fatal("expected error for unwritable directory")
	}
	if !strings.Contains(err.Error(), "write orders file") {
		t.Fatalf("expected descriptive error, got: %v", err)
	}
}

func TestApplyOrderRoutingDefaultsFillsMissingProviderModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Routing.Defaults.Provider = "codex"
	cfg.Routing.Defaults.Model = "gpt-5.3-codex"

	reg := ordersTestRegistry()
	of := OrdersFile{
		Orders: []Order{
			{
				ID:     "1",
				Status: OrderStatusActive,
				Stages: []Stage{
					{TaskKey: "execute", Status: StageStatusPending},
				},
			},
		},
	}

	got, changed := ApplyOrderRoutingDefaults(of, reg, cfg)
	if !changed {
		t.Fatal("expected changed")
	}
	if got.Orders[0].Stages[0].Provider != "codex" {
		t.Errorf("Provider = %q, want codex", got.Orders[0].Stages[0].Provider)
	}
	if got.Orders[0].Stages[0].Model != "gpt-5.3-codex" {
		t.Errorf("Model = %q, want gpt-5.3-codex", got.Orders[0].Stages[0].Model)
	}
}

func TestApplyOrderRoutingDefaultsFillsOnFailureStages(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Routing.Defaults.Provider = "claude"
	cfg.Routing.Defaults.Model = "claude-opus-4-6"

	reg := ordersTestRegistry()
	of := OrdersFile{
		Orders: []Order{
			{
				ID:     "1",
				Status: OrderStatusFailing,
				Stages: []Stage{
					{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusFailed},
				},
				OnFailure: []Stage{
					{TaskKey: "execute", Status: StageStatusPending},
				},
			},
		},
	}

	got, changed := ApplyOrderRoutingDefaults(of, reg, cfg)
	if !changed {
		t.Fatal("expected changed for OnFailure stage defaults")
	}
	if got.Orders[0].OnFailure[0].Provider != "claude" {
		t.Errorf("OnFailure Provider = %q, want claude", got.Orders[0].OnFailure[0].Provider)
	}
	if got.Orders[0].OnFailure[0].Model != "claude-opus-4-6" {
		t.Errorf("OnFailure Model = %q, want claude-opus-4-6", got.Orders[0].OnFailure[0].Model)
	}
}

func TestApplyOrderRoutingDefaultsPreservesStageExtra(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Routing.Defaults.Provider = "claude"
	cfg.Routing.Defaults.Model = "claude-opus-4-6"

	reg := ordersTestRegistry()
	of := OrdersFile{
		Orders: []Order{
			{
				ID:     "1",
				Status: OrderStatusActive,
				Stages: []Stage{
					{
						TaskKey: "execute",
						Status:  StageStatusPending,
						Extra: map[string]json.RawMessage{
							"priority": json.RawMessage(`99`),
							"custom":   json.RawMessage(`{"deep":"value"}`),
						},
					},
				},
			},
		},
	}

	got, changed := ApplyOrderRoutingDefaults(of, reg, cfg)
	if !changed {
		t.Fatal("expected changed")
	}

	extra := got.Orders[0].Stages[0].Extra
	if string(extra["priority"]) != `99` {
		t.Errorf("Extra[priority] = %s, want 99", extra["priority"])
	}
	if string(extra["custom"]) != `{"deep":"value"}` {
		t.Errorf("Extra[custom] = %s", extra["custom"])
	}
}

func TestApplyOrderRoutingDefaultsPreservesExplicit(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Routing.Defaults.Provider = "codex"
	cfg.Routing.Defaults.Model = "gpt-5.3-codex"

	reg := ordersTestRegistry()
	of := OrdersFile{
		Orders: []Order{
			{
				ID:     "1",
				Status: OrderStatusActive,
				Stages: []Stage{
					{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
				},
			},
		},
	}

	_, changed := ApplyOrderRoutingDefaults(of, reg, cfg)
	if changed {
		t.Fatal("expected no change when provider/model already set")
	}
}

func TestApplyOrderRoutingDefaultsUsesTagPolicy(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Routing.Defaults.Provider = "codex"
	cfg.Routing.Defaults.Model = "gpt-5.3-codex"
	cfg.Routing.Tags["execute"] = config.ModelPolicy{
		Provider: "claude",
		Model:    "claude-opus-4-6",
	}

	reg := ordersTestRegistry()
	of := OrdersFile{
		Orders: []Order{
			{
				ID:     "1",
				Status: OrderStatusActive,
				Stages: []Stage{
					{TaskKey: "execute", Status: StageStatusPending},
				},
			},
		},
	}

	got, changed := ApplyOrderRoutingDefaults(of, reg, cfg)
	if !changed {
		t.Fatal("expected changed")
	}
	// Tag policy should take precedence over defaults.
	if got.Orders[0].Stages[0].Provider != "claude" {
		t.Errorf("Provider = %q, want claude (from tag)", got.Orders[0].Stages[0].Provider)
	}
	if got.Orders[0].Stages[0].Model != "claude-opus-4-6" {
		t.Errorf("Model = %q, want claude-opus-4-6 (from tag)", got.Orders[0].Stages[0].Model)
	}
}

func TestNormalizeAndValidateOrdersDropsUnknownTaskTypes(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := ordersTestRegistry()
	of := OrdersFile{
		Orders: []Order{
			{
				ID:     "1",
				Status: OrderStatusActive,
				Stages: []Stage{
					{TaskKey: "nonexistent", Status: StageStatusPending},
					{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
				},
			},
		},
	}

	got, changed, err := NormalizeAndValidateOrders(of, nil, reg, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed")
	}
	if len(got.Orders) != 1 {
		t.Fatalf("Orders len = %d, want 1", len(got.Orders))
	}
	if len(got.Orders[0].Stages) != 1 {
		t.Fatalf("Stages len = %d, want 1", len(got.Orders[0].Stages))
	}
	if got.Orders[0].Stages[0].TaskKey != "execute" {
		t.Errorf("remaining stage TaskKey = %q", got.Orders[0].Stages[0].TaskKey)
	}
}

func TestNormalizeAndValidateOrdersRejectsDuplicateIDs(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := ordersTestRegistry()
	of := OrdersFile{
		Orders: []Order{
			{
				ID:     "1",
				Status: OrderStatusActive,
				Stages: []Stage{{TaskKey: "execute", Status: StageStatusPending}},
			},
			{
				ID:     "1",
				Status: OrderStatusActive,
				Stages: []Stage{{TaskKey: "execute", Status: StageStatusPending}},
			},
		},
	}

	_, _, err := NormalizeAndValidateOrders(of, nil, reg, cfg)
	if err == nil || !strings.Contains(err.Error(), "appears more than once") {
		t.Fatalf("expected duplicate ID error, got: %v", err)
	}
}

func TestNormalizeAndValidateOrdersRejectsEmptyStatus(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := ordersTestRegistry()
	of := OrdersFile{
		Orders: []Order{
			{
				ID:     "1",
				Status: "",
				Stages: []Stage{{TaskKey: "execute", Status: StageStatusPending}},
			},
		},
	}

	_, _, err := NormalizeAndValidateOrders(of, nil, reg, cfg)
	if err == nil {
		t.Fatal("expected error for empty status")
	}
	if !strings.Contains(err.Error(), "order status is required") {
		t.Fatalf("expected descriptive error, got: %v", err)
	}
}

func TestNormalizeAndValidateOrdersFailingEmptyOnFailureTerminal(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := ordersTestRegistry()
	of := OrdersFile{
		Orders: []Order{
			{
				ID:     "stuck",
				Status: OrderStatusFailing,
				Stages: []Stage{
					{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusFailed},
				},
				OnFailure: []Stage{
					{TaskKey: "nonexistent", Status: StageStatusPending},
				},
			},
		},
	}

	got, changed, err := NormalizeAndValidateOrders(of, nil, reg, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed")
	}
	if len(got.Orders) != 0 {
		t.Fatalf("expected order removed as terminal, got %d orders", len(got.Orders))
	}
	if len(got.ActionNeeded) == 0 || !strings.Contains(got.ActionNeeded[0], "terminal") {
		t.Fatalf("expected terminal annotation, got: %v", got.ActionNeeded)
	}
}

func TestNormalizeAndValidateOrdersValidatesOnFailureTaskTypes(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := ordersTestRegistry()

	t.Run("strips invalid OnFailure but keeps order", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				{
					ID:     "1",
					Status: OrderStatusActive,
					Stages: []Stage{
						{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
					},
					OnFailure: []Stage{
						{TaskKey: "nonexistent", Status: StageStatusPending},
						{TaskKey: "execute", Prompt: "valid fallback", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
					},
				},
			},
		}

		got, changed, err := NormalizeAndValidateOrders(of, nil, reg, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !changed {
			t.Fatal("expected changed")
		}
		if len(got.Orders) != 1 {
			t.Fatalf("Orders len = %d, want 1", len(got.Orders))
		}
		if len(got.Orders[0].OnFailure) != 1 {
			t.Fatalf("OnFailure len = %d, want 1", len(got.Orders[0].OnFailure))
		}
	})

	t.Run("all OnFailure invalid on active order clears OnFailure", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				{
					ID:     "2",
					Status: OrderStatusActive,
					Stages: []Stage{
						{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
					},
					OnFailure: []Stage{
						{TaskKey: "nonexistent", Status: StageStatusPending},
					},
				},
			},
		}

		got, changed, err := NormalizeAndValidateOrders(of, nil, reg, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !changed {
			t.Fatal("expected changed")
		}
		if len(got.Orders) != 1 {
			t.Fatalf("order should remain (active, not failing), got %d", len(got.Orders))
		}
		if got.Orders[0].OnFailure != nil {
			t.Errorf("OnFailure = %v, want nil", got.Orders[0].OnFailure)
		}
	})
}

func TestNormalizeAndValidateOrdersDropsEmptyStages(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := ordersTestRegistry()
	of := OrdersFile{
		Orders: []Order{
			{
				ID:     "empty-stages",
				Status: OrderStatusActive,
				Stages: []Stage{},
			},
		},
	}

	got, changed, err := NormalizeAndValidateOrders(of, nil, reg, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed")
	}
	if len(got.Orders) != 0 {
		t.Fatalf("expected order dropped, got %d", len(got.Orders))
	}
}

func TestNormalizeAndValidateOrdersDropsOnlyOnFailureStages(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := ordersTestRegistry()
	of := OrdersFile{
		Orders: []Order{
			{
				ID:     "only-onfailure",
				Status: OrderStatusActive,
				Stages: []Stage{},
				OnFailure: []Stage{
					{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
				},
			},
		},
	}

	got, changed, err := NormalizeAndValidateOrders(of, nil, reg, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed")
	}
	if len(got.Orders) != 0 {
		t.Fatalf("expected order dropped (no main stages), got %d", len(got.Orders))
	}
}

// jsonEqual compares two json.RawMessage values semantically.
func jsonEqual(t *testing.T, a, b json.RawMessage) bool {
	t.Helper()
	var va, vb interface{}
	if err := json.Unmarshal(a, &va); err != nil {
		t.Fatalf("unmarshal a: %v", err)
	}
	if err := json.Unmarshal(b, &vb); err != nil {
		t.Fatalf("unmarshal b: %v", err)
	}
	ra, _ := json.Marshal(va)
	rb, _ := json.Marshal(vb)
	return string(ra) == string(rb)
}

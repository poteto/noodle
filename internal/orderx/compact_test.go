package orderx

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestCompactParseSimpleThreeStageOrder(t *testing.T) {
	data := []byte(`{
		"orders": [
			{
				"id": "49",
				"title": "test order",
				"stages": [
					{"do": "execute", "with": "codex", "model": "gpt-5.3-codex"},
					{"do": "quality", "with": "claude", "model": "claude-opus-4-6"},
					{"do": "reflect", "with": "claude", "model": "claude-opus-4-6"}
				]
			}
		]
	}`)

	got, err := ParseCompactOrders(data)
	if err != nil {
		t.Fatalf("ParseCompactOrders returned error: %v", err)
	}

	want := CompactOrdersFile{
		Orders: []CompactOrder{
			{
				ID:    "49",
				Title: "test order",
				Stages: []CompactStage{
					{Do: "execute", With: "codex", Model: "gpt-5.3-codex"},
					{Do: "quality", With: "claude", Model: "claude-opus-4-6"},
					{Do: "reflect", With: "claude", Model: "claude-opus-4-6"},
				},
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseCompactOrders mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestCompactParseWithOptionalFields(t *testing.T) {
	data := []byte(`{
		"orders": [
			{
				"id": "optional",
				"stages": [
					{
						"do": "execute",
						"with": "codex",
						"model": "gpt-5.3-codex",
						"runtime": "sprites",
						"prompt": "Run implementation",
						"extra_prompt": "Focus on tests",
						"extra": {"meta": {"source": "schedule"}},
						"group": 2
					}
				]
			}
		]
	}`)

	got, err := ParseCompactOrders(data)
	if err != nil {
		t.Fatalf("ParseCompactOrders returned error: %v", err)
	}

	if len(got.Orders) != 1 || len(got.Orders[0].Stages) != 1 {
		t.Fatalf("unexpected parsed order shape: %#v", got)
	}
	stage := got.Orders[0].Stages[0]
	if stage.Do != "execute" {
		t.Fatalf("Do = %q, want %q", stage.Do, "execute")
	}
	if stage.With != "codex" {
		t.Fatalf("With = %q, want %q", stage.With, "codex")
	}
	if stage.Model != "gpt-5.3-codex" {
		t.Fatalf("Model = %q, want %q", stage.Model, "gpt-5.3-codex")
	}
	if stage.Runtime != "sprites" {
		t.Fatalf("Runtime = %q, want %q", stage.Runtime, "sprites")
	}
	if stage.Prompt != "Run implementation" {
		t.Fatalf("Prompt = %q, want %q", stage.Prompt, "Run implementation")
	}
	if stage.ExtraPrompt != "Focus on tests" {
		t.Fatalf("ExtraPrompt = %q, want %q", stage.ExtraPrompt, "Focus on tests")
	}
	if stage.Group != 2 {
		t.Fatalf("Group = %d, want %d", stage.Group, 2)
	}
	if !rawJSONEqual(stage.Extra["meta"], raw(`{"source":"schedule"}`)) {
		t.Fatalf("Extra[meta] mismatch: got %s", stage.Extra["meta"])
	}
}

func TestCompactParseWithActionNeeded(t *testing.T) {
	data := []byte(`{
		"orders": [
			{
				"id": "1",
				"stages": [
					{"do": "execute", "with": "codex", "model": "gpt-5.3-codex"}
				]
			}
		],
		"action_needed": ["update config", "confirm model"]
	}`)

	got, err := ParseCompactOrders(data)
	if err != nil {
		t.Fatalf("ParseCompactOrders returned error: %v", err)
	}

	want := []string{"update config", "confirm model"}
	if !reflect.DeepEqual(got.ActionNeeded, want) {
		t.Fatalf("ActionNeeded mismatch\n got: %#v\nwant: %#v", got.ActionNeeded, want)
	}
}

func TestCompactParseRejectsUnknownFields(t *testing.T) {
	data := []byte(`{
		"orders": [
			{
				"id": "1",
				"stages": [
					{"do": "execute", "with": "codex", "model": "gpt-5.3-codex", "unknown": true}
				]
			}
		]
	}`)

	_, err := ParseCompactOrders(data)
	if err == nil {
		t.Fatalf("ParseCompactOrders succeeded for unknown field")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("error did not mention unknown field: %v", err)
	}
}

func TestCompactParseRejectsStageWithoutWith(t *testing.T) {
	data := []byte(`{
		"orders": [
			{
				"id": "1",
				"stages": [
					{"do": "execute", "with": "", "model": "gpt-5.3-codex"}
				]
			}
		]
	}`)

	_, err := ParseCompactOrders(data)
	if err == nil {
		t.Fatalf("ParseCompactOrders succeeded with empty with")
	}
	if !strings.Contains(err.Error(), "provider is empty") {
		t.Fatalf("error did not mention provider empty: %v", err)
	}
}

func TestCompactParseRejectsStageWithoutModel(t *testing.T) {
	data := []byte(`{
		"orders": [
			{
				"id": "1",
				"stages": [
					{"do": "execute", "with": "codex", "model": ""}
				]
			}
		]
	}`)

	_, err := ParseCompactOrders(data)
	if err == nil {
		t.Fatalf("ParseCompactOrders succeeded with empty model")
	}
	if !strings.Contains(err.Error(), "model is empty") {
		t.Fatalf("error did not mention model empty: %v", err)
	}
}

func TestCompactParseAdHocStageIsValid(t *testing.T) {
	data := []byte(`{
		"orders": [
			{
				"id": "adhoc",
				"stages": [
					{"with": "claude", "model": "claude-opus-4-6", "prompt": "Investigate bug"}
				]
			}
		]
	}`)

	got, err := ParseCompactOrders(data)
	if err != nil {
		t.Fatalf("ParseCompactOrders returned error: %v", err)
	}

	if len(got.Orders) != 1 || len(got.Orders[0].Stages) != 1 {
		t.Fatalf("unexpected parsed order shape: %#v", got)
	}
	stage := got.Orders[0].Stages[0]
	if stage.Do != "" || stage.Prompt != "Investigate bug" {
		t.Fatalf("unexpected ad-hoc stage fields: %#v", stage)
	}
}

func TestCompactParseRejectsStageWithNeitherDoNorPrompt(t *testing.T) {
	data := []byte(`{
		"orders": [
			{
				"id": "1",
				"stages": [
					{"with": "codex", "model": "gpt-5.3-codex"}
				]
			}
		]
	}`)

	_, err := ParseCompactOrders(data)
	if err == nil {
		t.Fatalf("ParseCompactOrders succeeded with empty do and prompt")
	}
	if !strings.Contains(err.Error(), "task key and prompt are both empty") {
		t.Fatalf("error did not mention empty task key and prompt: %v", err)
	}
}

func TestCompactExpandSimpleOrderStatuses(t *testing.T) {
	compact := CompactOrdersFile{
		Orders: []CompactOrder{
			{
				ID: "49",
				Stages: []CompactStage{
					{Do: "execute", With: "codex", Model: "gpt-5.3-codex"},
					{Do: "quality", With: "claude", Model: "claude-opus-4-6"},
					{Do: "reflect", With: "claude", Model: "claude-opus-4-6"},
				},
			},
		},
	}

	got, err := ExpandCompactOrders(compact)
	if err != nil {
		t.Fatalf("ExpandCompactOrders returned error: %v", err)
	}

	if len(got.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(got.Orders))
	}
	if got.Orders[0].Status != OrderStatusActive {
		t.Fatalf("order status = %q, want %q", got.Orders[0].Status, OrderStatusActive)
	}
	for i, stage := range got.Orders[0].Stages {
		if stage.Status != StageStatusPending {
			t.Fatalf("stage %d status = %q, want %q", i, stage.Status, StageStatusPending)
		}
	}
}

func TestCompactExpandPreservesPassThroughFields(t *testing.T) {
	compact := CompactOrdersFile{
		Orders: []CompactOrder{
			{
				ID:        "pass-through",
				Title:     "pass-through title",
				Plan:      []string{"a", "b"},
				Rationale: "because",
				Stages: []CompactStage{
					{
						Do:          "execute",
						With:        "codex",
						Model:       "gpt-5.3-codex",
						Prompt:      "build it",
						ExtraPrompt: "plus tests",
						Extra: map[string]json.RawMessage{
							"meta": raw(`{"k":"v"}`),
						},
						Group: 3,
					},
				},
			},
		},
	}

	got, err := ExpandCompactOrders(compact)
	if err != nil {
		t.Fatalf("ExpandCompactOrders returned error: %v", err)
	}

	order := got.Orders[0]
	if order.Title != "pass-through title" {
		t.Fatalf("Title = %q, want %q", order.Title, "pass-through title")
	}
	if !reflect.DeepEqual(order.Plan, []string{"a", "b"}) {
		t.Fatalf("Plan mismatch: %#v", order.Plan)
	}
	if order.Rationale != "because" {
		t.Fatalf("Rationale = %q, want %q", order.Rationale, "because")
	}

	stage := order.Stages[0]
	if stage.Prompt != "build it" {
		t.Fatalf("Prompt = %q, want %q", stage.Prompt, "build it")
	}
	if stage.ExtraPrompt != "plus tests" {
		t.Fatalf("ExtraPrompt = %q, want %q", stage.ExtraPrompt, "plus tests")
	}
	if !reflect.DeepEqual(stage.Extra, map[string]json.RawMessage{"meta": raw(`{"k":"v"}`)}) {
		t.Fatalf("Extra mismatch: %#v", stage.Extra)
	}
	if stage.Group != 3 {
		t.Fatalf("Group = %d, want %d", stage.Group, 3)
	}
}

func TestCompactExpandSetsSkillFromDo(t *testing.T) {
	compact := CompactOrdersFile{
		Orders: []CompactOrder{
			{
				ID: "skill",
				Stages: []CompactStage{
					{Do: "execute", With: "codex", Model: "gpt-5.3-codex"},
				},
			},
		},
	}

	got, err := ExpandCompactOrders(compact)
	if err != nil {
		t.Fatalf("ExpandCompactOrders returned error: %v", err)
	}

	stage := got.Orders[0].Stages[0]
	if stage.Skill != "execute" {
		t.Fatalf("Skill = %q, want %q", stage.Skill, "execute")
	}
	if stage.TaskKey != "execute" {
		t.Fatalf("TaskKey = %q, want %q", stage.TaskKey, "execute")
	}
}

func TestCompactExpandAdHocStageLeavesTaskKeyAndSkillEmpty(t *testing.T) {
	compact := CompactOrdersFile{
		Orders: []CompactOrder{
			{
				ID: "adhoc",
				Stages: []CompactStage{
					{With: "claude", Model: "claude-opus-4-6", Prompt: "Investigate"},
				},
			},
		},
	}

	got, err := ExpandCompactOrders(compact)
	if err != nil {
		t.Fatalf("ExpandCompactOrders returned error: %v", err)
	}

	stage := got.Orders[0].Stages[0]
	if stage.TaskKey != "" {
		t.Fatalf("TaskKey = %q, want empty", stage.TaskKey)
	}
	if stage.Skill != "" {
		t.Fatalf("Skill = %q, want empty", stage.Skill)
	}
}

func TestCompactExpandOrderWithoutRuntimeLeavesRuntimeEmpty(t *testing.T) {
	compact := CompactOrdersFile{
		Orders: []CompactOrder{
			{
				ID: "runtime",
				Stages: []CompactStage{
					{Do: "execute", With: "codex", Model: "gpt-5.3-codex"},
				},
			},
		},
	}

	got, err := ExpandCompactOrders(compact)
	if err != nil {
		t.Fatalf("ExpandCompactOrders returned error: %v", err)
	}

	if got.Orders[0].Stages[0].Runtime != "" {
		t.Fatalf("Runtime = %q, want empty", got.Orders[0].Stages[0].Runtime)
	}
}

func TestCompactExpandEmptyOrdersArray(t *testing.T) {
	compact := CompactOrdersFile{Orders: []CompactOrder{}}

	got, err := ExpandCompactOrders(compact)
	if err != nil {
		t.Fatalf("ExpandCompactOrders returned error: %v", err)
	}

	if len(got.Orders) != 0 {
		t.Fatalf("Orders length = %d, want 0", len(got.Orders))
	}
	if got.Orders == nil {
		t.Fatalf("Orders is nil, want empty slice")
	}
}

func TestCompactParseEmptyWhitespaceInput(t *testing.T) {
	tests := []string{"", "   \n\t  "}
	for _, tc := range tests {
		got, err := ParseCompactOrders([]byte(tc))
		if err != nil {
			t.Fatalf("ParseCompactOrders returned error for %q: %v", tc, err)
		}
		if !reflect.DeepEqual(got, CompactOrdersFile{}) {
			t.Fatalf("ParseCompactOrders(%q) = %#v, want empty CompactOrdersFile", tc, got)
		}
	}
}

func raw(s string) json.RawMessage {
	return json.RawMessage(s)
}

func rawJSONEqual(got, want json.RawMessage) bool {
	var gotValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		return false
	}

	var wantValue any
	if err := json.Unmarshal(want, &wantValue); err != nil {
		return false
	}

	return reflect.DeepEqual(gotValue, wantValue)
}

package fixturedir

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type ErrorExpectation struct {
	Any      bool   `json:"any,omitempty"`
	Contains string `json:"contains,omitempty"`
	Equals   string `json:"equals,omitempty"`
	Absent   bool   `json:"absent,omitempty"`
}

type CountExpectation struct {
	Eq  *int `json:"eq,omitempty"`
	Min *int `json:"min,omitempty"`
	Max *int `json:"max,omitempty"`
}

type StringExpectation struct {
	Equals   string `json:"equals,omitempty"`
	Contains string `json:"contains,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
}

func AssertError(tb testing.TB, context string, err error, expect *ErrorExpectation) {
	tb.Helper()

	if strings.TrimSpace(context) == "" {
		context = "fixture"
	}
	if expect == nil {
		if err != nil {
			tb.Fatalf("%s error: %v", context, err)
		}
		return
	}
	if expect.Absent {
		if err != nil {
			tb.Fatalf("%s unexpected error: %v", context, err)
		}
		return
	}
	if err == nil {
		tb.Fatalf("%s expected error", context)
	}
	if want := strings.TrimSpace(expect.Equals); want != "" && err.Error() != want {
		tb.Fatalf("%s error = %q, want exact %q", context, err.Error(), want)
	}
	if want := strings.TrimSpace(expect.Contains); want != "" && !strings.Contains(err.Error(), want) {
		tb.Fatalf("%s error = %q, want contains %q", context, err.Error(), want)
	}
}

func AssertCounts(tb testing.TB, label string, got map[string]int, want map[string]CountExpectation) {
	tb.Helper()
	for key, expect := range want {
		actual, ok := got[key]
		if !ok {
			tb.Fatalf("%s missing count key %q", label, key)
		}
		if expect.Eq != nil && actual != *expect.Eq {
			tb.Fatalf("%s[%s] = %d, want eq %d", label, key, actual, *expect.Eq)
		}
		if expect.Min != nil && actual < *expect.Min {
			tb.Fatalf("%s[%s] = %d, want min %d", label, key, actual, *expect.Min)
		}
		if expect.Max != nil && actual > *expect.Max {
			tb.Fatalf("%s[%s] = %d, want max %d", label, key, actual, *expect.Max)
		}
	}
}

func AssertBools(tb testing.TB, label string, got map[string]bool, want map[string]bool) {
	tb.Helper()
	for key, expected := range want {
		actual, ok := got[key]
		if !ok {
			tb.Fatalf("%s missing bool key %q", label, key)
		}
		if actual != expected {
			tb.Fatalf("%s[%s] = %v, want %v", label, key, actual, expected)
		}
	}
}

func AssertStrings(tb testing.TB, label string, got map[string]string, want map[string]StringExpectation) {
	tb.Helper()
	for key, expected := range want {
		actual, ok := got[key]
		if !ok {
			tb.Fatalf("%s missing string key %q", label, key)
		}
		if value := strings.TrimSpace(expected.Equals); value != "" && actual != value {
			tb.Fatalf("%s[%s] = %q, want equals %q", label, key, actual, value)
		}
		if value := strings.TrimSpace(expected.Contains); value != "" && !strings.Contains(actual, value) {
			tb.Fatalf("%s[%s] = %q, want contains %q", label, key, actual, value)
		}
		if value := strings.TrimSpace(expected.Prefix); value != "" && !strings.HasPrefix(actual, value) {
			tb.Fatalf("%s[%s] = %q, want prefix %q", label, key, actual, value)
		}
	}
}

func AssertSequence[T comparable](tb testing.TB, label string, got, want []T) {
	tb.Helper()
	if !reflect.DeepEqual(got, want) {
		tb.Fatalf("%s = %v, want %v", label, got, want)
	}
}

func Keys[V any](input map[string]V) []string {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func DebugMap[V any](input map[string]V) string {
	keys := Keys(input)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, input[key]))
	}
	return strings.Join(parts, ", ")
}

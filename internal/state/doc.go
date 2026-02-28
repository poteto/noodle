// Package state defines the canonical backend state model.
//
// All loop scheduling decisions derive from this single unified model.
// Types cover orders, stages, attempts, lifecycle statuses, and run mode.
// Access-pattern indexes are computed from state, not stored separately.
package state

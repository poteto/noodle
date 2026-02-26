# Respect API Contracts

When decomposing work into parallel agents, identify **API contract boundaries** — where one phase defines a type/interface that another phase consumes.

## The Rule

**Never parallelize phases that share an API contract.** Run the producer first, then the consumer.

## Why

When Phase A defines a model and Phase B writes code against that model, running them in parallel means each invents its own version of the contract. The consumer's work becomes invalid when the producer's actual output doesn't match assumptions.

## Detection

Before spawning parallel agents, ask: "Does any phase's output become another phase's input?" If yes, they must be sequential — or use a single agent that owns both sides of the contract.

## Exceptions

- Phases that share a codebase but touch different files with no type-level coupling can still be parallel.
- If the API contract is already frozen (types exist and won't change), consumers can run in parallel.

## Relationship to Principles

This is [[principles/foundational-thinking|sequencing for option value]] applied to parallel dispatch — the producer's API is a structural decision ([[principles/foundational-thinking#data-structures-first]]) that shapes all downstream work, so it must stabilize before consumers start.

See also [[principles/serialize-shared-state-mutations]]

> **Stale:** This document predates Plan 23 (task-type skills) and Plan 25 (TUI revamp). Terminology (sous-chef, taster, spawner, config path) is outdated. See `[[plans/23-task-type-skill-suite/overview]]` for current architecture. Rewrite tracked by todo #24.

# Noodle: Open-Source AI Coding Framework

## The Problem

The human is the scheduler for all project work: initiating audits, deciding what to plan, picking which plans to execute, triggering reflection. Every step blocks on human attention — the scarcest resource in the system.

## The Vision

Noodle replaces the human as the top-level scheduler. It continuously drives a project's backlog — planning, executing, verifying, reflecting — while the human observes and intervenes only when they choose to. The system is designed for full autonomy by default, capable of running unattended for hours.

Users extend Noodle by writing or overriding skills — pure Claude Code skills with no Noodle-specific metadata. All wiring lives in a single config file.

## Kitchen Brigade

All actor roles adopt kitchen terminology:

| Role | What | Actor |
|------|------|-------|
| **Chef** | Human — sets strategy, intervenes when they choose | Human |
| **Sous Chef** | Scheduler — reads the mise, prioritizes the queue, decides what to cook next | LLM agent (configurable model) |
| **Taster** | Quality reviewer — checks every dish before it leaves the pass | LLM agent |
| **Cook** | Does the work — uses native sub-agents as needed | Claude or Codex session |
| **Mise** | Gathers state into a structured brief (mise en place) | Go code (not an actor) |

## Architecture

```
                     ┌─────────────────────┐
                     │     Chef (TUI)      │
                     │  observe · intervene │
                     └─────────┬───────────┘
                               │
┌──────────────────────────────┼──────────────────────────────┐
│                        noodle cook                          │
│                              │                              │
│  Mise (Go) ──► Sous Chef (LLM) ──► Prioritized Queue       │
│       ↓              ↓                    ↓                 │
│  Structured Brief  Scheduling judgment   Spawn decisions    │
│                                           ↓                 │
│              Cook Loop ◄──── Spawner (tmux/future cloud)    │
│              enforce constraints · spawn · monitor · log    │
│                      ↓                                      │
│              NDJSON Log (source of truth)                    │
│                      ↓                                      │
│              Bubbletea TUI (optional)                        │
└─────────────────────────────────────────────────────────────┘
                       │
           ┌───────────┴───────────┐
           │   Cook Sessions       │
           │   plan · execute      │
           │   verify · reflect    │
           │   quality · systemfix │
           └───────────────────────┘
```

## Key Principles

### Skills as the only extension point

Skills are pure Claude Code skills (`SKILL.md` + optional `references/`). No per-skill Noodle metadata. Users extend Noodle by writing or overriding skills, wired in via `.noodle/config.toml`.

### LLM-powered prioritization

The mise (Go) gathers backlog state, resource state, active sessions, recent history into a structured brief. The sous chef (LLM) reads the brief, applies scheduling judgment, outputs a prioritized queue. The Go loop reads the queue mechanically and enforces hard constraints.

### Deterministic execution

The Go loop never makes judgment calls. Judgment lives in the sous chef (LLM). The Go loop reads the queue and enforces hard constraints mechanically.

### The log is the source of truth

Every cycle produces an NDJSON entry: session ID, phase, target, outcome, cost, commit hashes, concurrent sessions, decision trace. The TUI, CLI status commands, and future web UI all read from this log.

### Autonomy is a dial, not a switch

The chef tunes trust from full headless autonomy to manual approval of every spawn decision.

### Self-recovering by default

Session crashes auto-retry. Spawn failures retry with backoff. The systemfix phase spawns an agent to diagnose root cause before continuing.

## What Ships with Noodle

**Tier 1 — Go binary (lean core):**
- Mise (data gathering, brief construction)
- Cook loop (read queue, enforce constraints, spawn, monitor, log)
- Spawner (tmux today, Spawner interface for future cloud)
- TUI, CLI
- Skill resolver (project > user > bundled precedence)
- Adapter runner (execute sync scripts, read normalized output)

**Tier 2 — Default skills (fully overridable):**
- `sous-chef` — scheduling and prioritization
- `taster` — quality review
- `backlog` — teaches agents to read/write the backlog
- `plans` — teaches agents to read/write plans
- `noodle` — meta-skill: how to configure and extend Noodle
- `bootstrap` — detects missing infra, scaffolds defaults

**Tier 3 — Default adapters (sync scripts):**
- `sync-backlog.sh` — parses backlog to normalized NDJSON
- `sync-plans.sh` — parses plans to normalized NDJSON

## Current Rollout Posture

- Prelaunch posture (2026-02-21): prioritize clean contracts and hard cutovers over backwards compatibility. No external users yet, so legacy-support complexity is optional.

## Principles Applied

- [[principles/never-block-on-the-human]] — the human supervises asynchronously; agents stay unblocked
- [[principles/encode-lessons-in-structure]] — CLI enforces process, prompts focus on work
- [[principles/cost-aware-delegation]] — LLM for judgment, Go for mechanics, resource-aware parallelism
- [[principles/observe-directly]] — the log is the source of truth; direct process liveness checks
- [[principles/redesign-from-first-principles]] — rewrite over refactor when 86% of code is coupled to the old model

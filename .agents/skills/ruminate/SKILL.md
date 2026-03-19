---
name: ruminate
description: >-
  Mine past Claude Code and Codex conversations for uncaptured patterns, corrections, and knowledge.
  Cross-references with existing brain content. Triggers: "ruminate", "mine my history",
  "what have I been working on", "review past sessions", "extract learnings".
---

# Ruminate

Mine conversation history for brain-worthy knowledge that was never captured. Complements `reflect` (current session) and `meditate` (brain vault audit) by looking at the full archive of past conversations across both providers.

## Process

**Use Tasks to track progress.** Create a task for each step below (TaskCreate), mark each in_progress when starting and completed when done (TaskUpdate). Check TaskList after each step.

### 1. Read the brain

Build a brain snapshot: `sh .claude/skills/meditate/scripts/snapshot.sh brain/ /tmp/brain-snapshot-ruminate.md`. Pass the snapshot path to each analysis agent. This avoids loading the full brain into the ruminate orchestrator's context.

### 2. Locate conversations

Find both provider roots:

1. Claude project directory:
   `~/.claude/projects/-<cwd-with-dashes-replacing-slashes>/`
2. Codex sessions root:
   `~/.codex/sessions/`

For example, `/Users/lauren/code/noodle` maps to
`~/.claude/projects/-Users-lauren-code-noodle/` for Claude and uses `~/.codex/sessions/` for Codex.

### 3. Extract conversations

Run the extraction script to parse both JSONL formats into readable text and split into batches:

```bash
SKILL_DIR="$(dirname "$(realpath "$0")")/.."  # adjust path as needed
CLAUDE_DIR="$HOME/.claude/projects/-<project-slug>"
CODEX_DIR="$HOME/.codex/sessions"
OUT_DIR="/tmp/ruminate-$(date +%s)"

python3 "$SKILL_DIR/scripts/extract-conversations.py" "$OUT_DIR" \
  --claude-dir "$CLAUDE_DIR" \
  --codex-dir "$CODEX_DIR" \
  --cwd "$PWD" \
  --batches N
```

Choose N based on total extracted conversations (Claude + Codex): ~1 batch per 20 conversations, minimum 2, maximum 10.

### 4. Spawn analysis team

Create an agent team (TeamCreate) with N agents (one per batch, matching the batch count from step 3), each with `subagent_type: general-purpose` and `model: opus`. Run all N in parallel.

Each agent's prompt should include:

- The batch manifest path (`$OUT_DIR/batches/batch_N.txt`)
- The output path (`$OUT_DIR/findings_N.md`)
- The list of topics **already captured** in the brain (compiled from step 1) — so agents skip known knowledge
- A reminder that each extracted file includes provider/source metadata headers (`[PROVIDER]`, `[CWD]`, `[SOURCE_FILE]`) and should be used as evidence context
- Instructions to extract from each conversation:
  - **User corrections**: times the user corrected the assistant's approach, code, or understanding
  - **Recurring preferences**: things the user explicitly asked for or pushed back on repeatedly
  - **Technical learnings**: codebase-specific knowledge, gotchas, patterns discovered
  - **Workflow patterns**: how the user prefers to work
  - **Frustrations**: friction points, wasted effort, things that went wrong
  - **Skills wished for**: capabilities the user expressed wanting

Agents write structured findings to their output files.

### 5. Synthesize

After all agents complete, read all findings files. Cross-reference with existing brain content. Deduplicate across batches.

**Filter by frequency and impact.** Most findings won't be worth adding. Apply these filters before presenting:

- **Frequency**: Did this come up in multiple conversations, or was the user correcting the same mistake repeatedly? One-off corrections are usually not worth a brain entry — the brain should capture *patterns*, not incidents.
- **Factual accuracy**: Is something in the brain now wrong? (e.g. a rule was disabled but the brain still documents it as active). These are always worth fixing regardless of frequency.
- **Impact**: Would failing to capture this cause repeated wasted effort in future sessions? A gotcha that cost 5 minutes once is low-impact. A pattern that caused 3 rounds of corrections is high-impact.

**Discard aggressively.** It's better to present 3 high-signal findings than 9 that include noise. If a finding only happened once and isn't a factual correction, skip it.

### 6. Present and apply

Present findings to the user in a table with columns: finding, frequency/evidence, and proposed action. Be honest about which findings are one-offs vs. recurring patterns — let the user decide what's worth adding.

**Route skill-specific learnings.** Check if any findings are about how a specific skill should work — its process, prompts, edge cases, or troubleshooting. Update the skill's SKILL.md or references/ directly. Read the skill first to avoid duplicating or contradicting existing content.

Apply only the changes the user approves. Follow brain writing conventions:

- One topic per file, organized in directories
- Use `[[wikilinks]]` to connect related notes
- Update `brain/index.md` after all changes
- Default to updating existing notes over creating new ones

### 7. Clean up

Remove the temporary extraction directory:

```bash
rm -rf "$OUT_DIR"
```

## Guidelines

- **Filter aggressively.** Most conversations will have low signal — automated tasks, trivial exchanges, already-captured knowledge. Only surface what's genuinely new and impactful.
- **Prefer reduction.** If a finding is a special case of an existing brain principle, update the existing note rather than creating a new one.
- **Quote the user.** When a finding stems from a direct user correction, include the user's words and source file path — they carry the most signal about what matters.
- **Shut down agents** when analysis is complete. Don't leave them idle.

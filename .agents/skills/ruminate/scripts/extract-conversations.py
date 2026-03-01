#!/usr/bin/env python3
"""Extract user and assistant messages from Claude and Codex conversation JSONL files.

Usage:
  extract-conversations.py <output-dir> --claude-dir DIR [--codex-dir DIR] [options]
  extract-conversations.py <output-dir> --codex-dir DIR [--claude-dir DIR] [options]

Options:
  --claude-dir DIR   Claude project conversation directory (repeatable)
  --codex-dir DIR    Codex sessions root directory (repeatable, recursive)
  --cwd PATH         Include only conversations from this working directory
  --batches N        Number of batch manifests to create (default: 5)
  --from YYYY-MM-DD  Include conversations modified on or after this date
  --to YYYY-MM-DD    Include conversations modified on or before this date
  --min-size BYTES   Minimum file size in bytes (default: 500)

Date filters are composable:
  --from 2026-02-13 --to 2026-02-13   Exactly Feb 13
  --from 2026-02-13                    Feb 13 onwards
  --to 2026-02-13                      Up to and including Feb 13

Output:
  <output-dir>/000_<provider>_<name>.txt  — extracted messages per conversation
  <output-dir>/batches/batch_0.txt ... batch_N.txt — file lists for each batch
"""

import argparse
import glob
import json
import os
import sys
from datetime import date, datetime
from typing import Any


def file_mod_date(fpath: str) -> date:
    """Return the modification date of a file."""
    return datetime.fromtimestamp(os.path.getmtime(fpath)).date()


def extract_claude_texts(content: Any) -> list[str]:
    texts: list[str] = []
    if isinstance(content, str):
        texts.append(content)
        return texts
    if not isinstance(content, list):
        return texts

    for item in content:
        if not isinstance(item, dict):
            continue
        if item.get("type") == "text" and isinstance(item.get("text"), str):
            texts.append(item["text"])
    return texts


def extract_codex_texts(content: Any) -> list[str]:
    texts: list[str] = []
    if isinstance(content, str):
        texts.append(content)
        return texts
    if not isinstance(content, list):
        return texts

    for item in content:
        if not isinstance(item, dict):
            continue
        item_type = item.get("type")
        if item_type in {"input_text", "output_text", "text"} and isinstance(item.get("text"), str):
            texts.append(item["text"])
    return texts


def append_message(messages: list[str], role: str, text: str, max_len: int) -> None:
    clean = text.strip()
    if not clean or len(clean) <= 10:
        return
    if clean.startswith("<system-reminder>") and clean.endswith("</system-reminder>"):
        return
    if role == "user":
        messages.append(f"[USER]: {text[:max_len]}")
        return
    if role == "assistant":
        messages.append(f"[ASSISTANT]: {text[:max_len]}")


def extract_messages(fpath: str) -> tuple[str, str, list[str]]:
    """Extract messages from one conversation JSONL file.

    Returns (provider, cwd, messages), where provider is one of:
    claude, codex, unknown.
    """
    provider = "unknown"
    cwd = ""
    messages: list[str] = []

    with open(fpath, encoding="utf-8", errors="replace") as handle:
        for line in handle:
            try:
                entry = json.loads(line)
            except (json.JSONDecodeError, ValueError):
                continue
            if not isinstance(entry, dict):
                continue

            entry_type = entry.get("type", "")

            if isinstance(entry.get("cwd"), str) and not cwd:
                cwd = entry["cwd"]
                provider = "claude"

            if entry_type == "session_meta":
                payload = entry.get("payload")
                if isinstance(payload, dict):
                    provider = "codex"
                    if isinstance(payload.get("cwd"), str) and not cwd:
                        cwd = payload["cwd"]
                continue

            if entry_type in {"user", "assistant"}:
                provider = "claude"
                if entry_type == "user" and entry.get("isMeta"):
                    continue
                msg = entry.get("message")
                if not isinstance(msg, dict):
                    continue
                for text in extract_claude_texts(msg.get("content", "")):
                    append_message(messages, entry_type, text, 3000 if entry_type == "user" else 800)
                continue

            if entry_type != "response_item":
                continue

            payload = entry.get("payload")
            if not isinstance(payload, dict):
                continue
            if payload.get("type") != "message":
                continue

            role = payload.get("role", "")
            if role not in {"user", "assistant"}:
                continue
            provider = "codex"
            for text in extract_codex_texts(payload.get("content", "")):
                append_message(messages, role, text, 3000 if role == "user" else 800)

    return provider, cwd, messages


def discover_files(claude_dirs: list[str], codex_dirs: list[str]) -> list[str]:
    files: list[str] = []
    seen: set[str] = set()

    for root in claude_dirs:
        for fpath in glob.glob(os.path.join(root, "*.jsonl")):
            real = os.path.realpath(fpath)
            if real not in seen:
                seen.add(real)
                files.append(real)

    for root in codex_dirs:
        for fpath in glob.glob(os.path.join(root, "**", "*.jsonl"), recursive=True):
            real = os.path.realpath(fpath)
            if real not in seen:
                seen.add(real)
                files.append(real)

    return files


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Extract messages from Claude and Codex conversation JSONL files."
    )
    parser.add_argument("output_dir", help="Directory to write extracted conversations")
    parser.add_argument("--claude-dir", action="append", default=[], help="Claude project dir containing .jsonl files")
    parser.add_argument("--codex-dir", action="append", default=[], help="Codex sessions root (searched recursively)")
    parser.add_argument("--cwd", help="Only include conversations where detected cwd matches this path")
    parser.add_argument("--batches", type=int, default=5, help="Number of batch manifests (default: 5)")
    parser.add_argument(
        "--from",
        dest="from_date",
        type=date.fromisoformat,
        help="Include conversations modified on or after this date (YYYY-MM-DD)",
    )
    parser.add_argument(
        "--to",
        dest="to_date",
        type=date.fromisoformat,
        help="Include conversations modified on or before this date (YYYY-MM-DD)",
    )
    parser.add_argument("--min-size", type=int, default=500, help="Minimum file size in bytes (default: 500)")
    args = parser.parse_args()

    if not args.claude_dir and not args.codex_dir:
        parser.error("at least one input source is required (--claude-dir or --codex-dir)")

    os.makedirs(args.output_dir, exist_ok=True)

    files = []
    for fpath in discover_files(args.claude_dir, args.codex_dir):
        if os.path.getsize(fpath) < args.min_size:
            continue
        if args.from_date or args.to_date:
            mod = file_mod_date(fpath)
            if args.from_date and mod < args.from_date:
                continue
            if args.to_date and mod > args.to_date:
                continue
        files.append(fpath)

    files.sort(key=os.path.getmtime, reverse=True)

    date_desc = ""
    if args.from_date and args.to_date:
        date_desc = f" (from {args.from_date} to {args.to_date})"
    elif args.from_date:
        date_desc = f" (from {args.from_date})"
    elif args.to_date:
        date_desc = f" (to {args.to_date})"
    print(f"Found {len(files)} candidate conversations{date_desc}", file=sys.stderr)

    extracted: list[str] = []
    provider_counts: dict[str, int] = {"claude": 0, "codex": 0, "unknown": 0}

    for idx, fpath in enumerate(files):
        provider, cwd, messages = extract_messages(fpath)
        if args.cwd and cwd and os.path.realpath(cwd) != os.path.realpath(args.cwd):
            continue
        if not messages:
            continue

        provider_counts[provider] = provider_counts.get(provider, 0) + 1
        fname = os.path.basename(fpath).replace(".jsonl", "")
        out_path = f"{args.output_dir}/{idx:03d}_{provider}_{fname}.txt"
        with open(out_path, "w", encoding="utf-8") as out:
            out.write(f"[PROVIDER]: {provider}\n")
            if cwd:
                out.write(f"[CWD]: {cwd}\n")
            out.write(f"[SOURCE_FILE]: {fpath}\n\n")
            out.write("\n\n".join(messages))
            out.write("\n")
        extracted.append(out_path)

    print(
        "Extracted "
        f"{len(extracted)} conversations with content "
        f"(claude={provider_counts.get('claude', 0)}, "
        f"codex={provider_counts.get('codex', 0)}, "
        f"unknown={provider_counts.get('unknown', 0)})",
        file=sys.stderr,
    )

    batch_dir = f"{args.output_dir}/batches"
    os.makedirs(batch_dir, exist_ok=True)
    batch_size = max(1, (len(extracted) + args.batches - 1) // args.batches)

    for b in range(args.batches):
        batch_files = extracted[b * batch_size : (b + 1) * batch_size]
        if not batch_files:
            continue
        manifest = f"{batch_dir}/batch_{b}.txt"
        with open(manifest, "w", encoding="utf-8") as mf:
            mf.write("\n".join(batch_files) + "\n")
        print(f"Batch {b}: {len(batch_files)} conversations", file=sys.stderr)

    print(args.output_dir)


if __name__ == "__main__":
    main()

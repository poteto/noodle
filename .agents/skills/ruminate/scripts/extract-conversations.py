#!/usr/bin/env python3
"""Extract user and assistant messages from Claude Code conversation JSONL files.

Usage: extract-conversations.py <project-dir> <output-dir> [options]

Options:
  --batches N        Number of batch manifests to create (default: 5)
  --from YYYY-MM-DD  Include conversations modified on or after this date
  --to YYYY-MM-DD    Include conversations modified on or before this date
  --min-size BYTES   Minimum file size in bytes (default: 500)

Date filters are composable:
  --from 2026-02-13 --to 2026-02-13   Exactly Feb 13
  --from 2026-02-13                    Feb 13 onwards
  --to 2026-02-13                      Up to and including Feb 13

Output:
  <output-dir>/000_<uuid>.txt  — extracted messages per conversation
  <output-dir>/batches/batch_0.txt ... batch_N.txt — file lists for each batch
"""

import argparse
import json
import os
import sys
import glob
from datetime import date, datetime


def extract_messages(fpath: str) -> list[str]:
    """Extract user and assistant text messages from a JSONL conversation file."""
    messages = []
    with open(fpath) as f:
        for line in f:
            try:
                d = json.loads(line)
            except (json.JSONDecodeError, ValueError):
                continue

            msg_type = d.get("type", "")
            msg = d.get("message", {})
            if not isinstance(msg, dict):
                continue

            content = msg.get("content", "")
            texts = []
            if isinstance(content, str):
                texts.append(content)
            elif isinstance(content, list):
                for c in content:
                    if isinstance(c, dict) and c.get("type") == "text":
                        texts.append(c["text"])

            for t in texts:
                clean = t.strip()
                if not clean or len(clean) <= 10:
                    continue
                # Skip system-reminder-only messages
                if clean.startswith("<system-reminder>") and clean.endswith("</system-reminder>"):
                    continue

                if msg_type == "user" and not d.get("isMeta"):
                    messages.append(f"[USER]: {t[:3000]}")
                elif msg_type == "assistant":
                    messages.append(f"[ASSISTANT]: {t[:800]}")

    return messages


def file_mod_date(fpath: str) -> date:
    """Return the modification date of a file."""
    return datetime.fromtimestamp(os.path.getmtime(fpath)).date()


def main():
    parser = argparse.ArgumentParser(
        description="Extract messages from Claude Code conversation JSONL files."
    )
    parser.add_argument("project_dir", help="Directory containing .jsonl conversation files")
    parser.add_argument("output_dir", help="Directory to write extracted conversations")
    parser.add_argument("--batches", type=int, default=5, help="Number of batch manifests (default: 5)")
    parser.add_argument("--from", dest="from_date", type=date.fromisoformat,
                        help="Include conversations modified on or after this date (YYYY-MM-DD)")
    parser.add_argument("--to", dest="to_date", type=date.fromisoformat,
                        help="Include conversations modified on or before this date (YYYY-MM-DD)")
    parser.add_argument("--min-size", type=int, default=500,
                        help="Minimum file size in bytes (default: 500)")

    args = parser.parse_args()

    os.makedirs(args.output_dir, exist_ok=True)

    # Find conversation files, applying size and date filters
    files = []
    for f in glob.glob(f"{args.project_dir}/*.jsonl"):
        if os.path.getsize(f) < args.min_size:
            continue
        if args.from_date or args.to_date:
            mod = file_mod_date(f)
            if args.from_date and mod < args.from_date:
                continue
            if args.to_date and mod > args.to_date:
                continue
        files.append(f)

    files.sort(key=os.path.getmtime, reverse=True)

    date_desc = ""
    if args.from_date and args.to_date:
        date_desc = f" (from {args.from_date} to {args.to_date})"
    elif args.from_date:
        date_desc = f" (from {args.from_date})"
    elif args.to_date:
        date_desc = f" (to {args.to_date})"
    print(f"Found {len(files)} non-empty conversations{date_desc}", file=sys.stderr)

    # Extract messages
    extracted = []
    for idx, fpath in enumerate(files):
        fname = os.path.basename(fpath).replace(".jsonl", "")
        out_path = f"{args.output_dir}/{idx:03d}_{fname}.txt"
        messages = extract_messages(fpath)
        if messages:
            with open(out_path, "w") as out:
                out.write("\n\n".join(messages))
            extracted.append(out_path)

    print(f"Extracted {len(extracted)} conversations with content", file=sys.stderr)

    # Create batch manifests
    batch_dir = f"{args.output_dir}/batches"
    os.makedirs(batch_dir, exist_ok=True)
    batch_size = max(1, (len(extracted) + args.batches - 1) // args.batches)

    for b in range(args.batches):
        batch_files = extracted[b * batch_size : (b + 1) * batch_size]
        if not batch_files:
            continue
        manifest = f"{batch_dir}/batch_{b}.txt"
        with open(manifest, "w") as mf:
            mf.write("\n".join(batch_files) + "\n")
        print(f"Batch {b}: {len(batch_files)} conversations", file=sys.stderr)

    # Print output dir for the caller
    print(args.output_dir)


if __name__ == "__main__":
    main()

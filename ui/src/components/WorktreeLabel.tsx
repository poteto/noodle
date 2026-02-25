import { middleTruncate } from "~/client";

export function WorktreeLabel({ name }: { name?: string }) {
  if (!name) return null;
  return (
    <span className="worktree-label" title={name}>
      {middleTruncate(name, 20)}
    </span>
  );
}

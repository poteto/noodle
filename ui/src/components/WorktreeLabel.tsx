import { middleTruncate } from "~/client";
import { Tooltip } from "./Tooltip";

export function WorktreeLabel({ name }: { name?: string }) {
  if (!name) return null;
  return (
    <Tooltip content={name}>
      <span className="worktree-label">
        {middleTruncate(name, 20)}
      </span>
    </Tooltip>
  );
}

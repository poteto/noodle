import { middleTruncate } from "~/client";
import { Tooltip } from "./Tooltip";

export function WorktreeLabel({ name }: { name?: string }) {
  if (!name) {
    return null;
  }
  return (
    <Tooltip content={name}>
      <span className="px-1.5 py-px bg-nblue-bg border border-nblue-dim text-[0.6875rem] text-nblue max-w-[140px] overflow-hidden text-ellipsis whitespace-nowrap">
        {middleTruncate(name, 20)}
      </span>
    </Tooltip>
  );
}

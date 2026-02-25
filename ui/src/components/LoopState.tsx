import type { LoopState as LoopStateType } from "~/client";

export function LoopState({ state }: { state: LoopStateType }) {
  return (
    <span className="loop-indicator">
      <span className="pulse-dot" />
      {state}
    </span>
  );
}

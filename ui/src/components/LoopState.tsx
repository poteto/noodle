import type { LoopState as LoopStateType, SSEStatus } from "~/client";
import { useSSEStatus } from "~/client";

export function LoopState({ state }: { state: LoopStateType }) {
  const sseStatus = useSSEStatus();
  const disconnected = sseStatus === "disconnected";
  const label = disconnected ? "disconnected" : state;

  return (
    <span className={`loop-indicator${disconnected ? " disconnected" : ""}`}>
      <span className={`pulse-dot${disconnected ? " dot-off" : ""}`} />
      {label}
    </span>
  );
}

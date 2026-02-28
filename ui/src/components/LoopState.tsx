import type { LoopState as LoopStateType } from "~/client";
import { useSSEStatus } from "~/client";

export function LoopState({ state }: { state: LoopStateType }) {
  const sseStatus = useSSEStatus();
  const disconnected = sseStatus === "disconnected";
  const label = disconnected ? "disconnected" : state;

  return (
    <span
      className={`flex items-center gap-[5px] px-2.5 py-1 font-semibold border ${
        disconnected ? "bg-red text-white border-red" : "bg-accent text-bg-depth border-accent"
      }`}
    >
      <span
        className={`w-1.5 h-1.5 ${
          disconnected ? "bg-white opacity-50" : "bg-bg-depth animate-pulse-dot"
        }`}
      />
      {label}
    </span>
  );
}

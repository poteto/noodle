import type { LoopState } from "~/client";
import { useSendControl } from "~/client";

export function LoopControls({ loopState }: { loopState: LoopState }) {
  const { mutate: send, isPending } = useSendControl();
  const isPaused = loopState === "paused";

  function toggle() {
    send({ action: isPaused ? "resume" : "pause" });
  }

  return (
    <button
      className={`loop-control-btn${isPaused ? " paused" : ""}`}
      onClick={toggle}
      disabled={isPending}
    >
      {isPaused ? "resume" : "pause"}
    </button>
  );
}

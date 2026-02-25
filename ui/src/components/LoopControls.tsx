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
      className={`px-5 py-2 font-mono text-[0.8125rem] font-bold border-2 cursor-pointer shadow-btn transition-[transform,box-shadow] duration-[0.12s] hover:-translate-x-0.5 hover:-translate-y-0.5 hover:shadow-btn-hover active:translate-x-px active:translate-y-px active:shadow-btn-active ${
        isPaused
          ? "bg-ngreen border-ngreen text-white"
          : "bg-accent border-border text-bg-0"
      }`}
      onClick={toggle}
      disabled={isPending}
    >
      {isPaused ? "resume" : "pause"}
    </button>
  );
}

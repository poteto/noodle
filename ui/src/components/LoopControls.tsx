import type { LoopState } from "~/client";
import { useControl } from "./ControlContext";

export function LoopControls({ loopState }: { loopState: LoopState }) {
  const send = useControl();
  const isPaused = loopState === "paused";

  function toggle() {
    send({ action: isPaused ? "resume" : "pause" });
  }

  return (
    <div className="group">
    <button
      className={`px-5 py-2 font-mono text-[0.8125rem] font-bold border-2 cursor-pointer shadow-btn transition-[transform,box-shadow] duration-[0.12s] group-hover:-translate-x-0.5 group-hover:-translate-y-0.5 group-hover:shadow-btn-hover active:translate-x-px active:translate-y-px active:shadow-btn-active ${
        isPaused
          ? "bg-ngreen border-ngreen text-white"
          : "bg-accent border-border text-bg-0"
      }`}
      onClick={toggle}
    >
      {isPaused ? "resume" : "pause"}
    </button>
    </div>
  );
}

import type { Snapshot } from "~/client";
import { LoopState } from "./LoopState";
import { LoopControls } from "./LoopControls";
import { StatBadge } from "./StatBadge";

export function BoardHeader({
  snapshot,
  onNewTask,
}: {
  snapshot: Snapshot;
  onNewTask: () => void;
}) {
  return (
    <div className="flex items-center justify-between px-10 pt-7 pb-[22px] border-b-3 border-border bg-bg-0 shrink-0">
      <div className="flex items-center gap-6">
        <h1 className="font-display font-extrabold text-[3.5rem] text-text-0 tracking-[-0.02em] leading-[0.85]">
          noodle
        </h1>
        <div className="flex items-center gap-2 font-mono text-[0.8125rem]">
          <LoopState state={snapshot.loop_state} />
          <StatBadge label="" value={`$${snapshot.total_cost_usd.toFixed(2)}`} />
        </div>
      </div>
      <div className="flex items-center gap-2.5">
        <LoopControls loopState={snapshot.loop_state} />
        <div className="group">
          <button
            type="button"
            className="flex items-center gap-1.5 px-6 py-2 bg-accent text-bg-0 font-display text-[0.9375rem] font-bold tracking-[0.04em] border-2 border-border shadow-btn cursor-pointer transition-[transform,box-shadow] duration-[0.12s] group-hover:-translate-x-0.5 group-hover:-translate-y-0.5 group-hover:shadow-btn-hover active:translate-x-px active:translate-y-px active:shadow-btn-active"
            onClick={onNewTask}
          >
            + new task
          </button>
        </div>
      </div>
    </div>
  );
}

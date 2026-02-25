import type { Snapshot, LoopState as LoopStateType } from "~/client";
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
  const doneCount = snapshot.recent.filter((s) => s.status !== "failed").length;
  const failedCount = snapshot.recent.filter((s) => s.status === "failed").length;

  return (
    <div className="board-header">
      <div className="board-header-left">
        <h1 className="board-title">noodle</h1>
        <div className="board-stats">
          <LoopState state={snapshot.loop_state as LoopStateType} />
          <StatBadge label="cooking" value={snapshot.active.length} />
          {snapshot.pending_review_count > 0 && (
            <StatBadge label="review" value={snapshot.pending_review_count} />
          )}
          <StatBadge label="done" value={doneCount} />
          {failedCount > 0 && (
            <StatBadge label="failed" value={failedCount} />
          )}
          <StatBadge label="" value={`$${snapshot.total_cost_usd.toFixed(2)}`} />
        </div>
      </div>
      <div className="board-header-right">
        <LoopControls loopState={snapshot.loop_state as LoopStateType} />
        <button className="new-task-btn" onClick={onNewTask}>
          + new task
        </button>
      </div>
    </div>
  );
}

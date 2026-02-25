import type { Session } from "~/client";
import { useSendControl, formatCost } from "~/client";
import { Badge } from "./Badge";
import { WorktreeLabel } from "./WorktreeLabel";
import { RotateCcw } from "lucide-react";

export function DoneCard({ session }: { session: Session }) {
  const { mutate: send, isPending } = useSendControl();
  const failed = session.status === "failed";
  const taskKey = session.task_key ?? "";

  function handleReplay(e: React.MouseEvent) {
    e.stopPropagation();
    if (failed) {
      send({ action: "requeue", item: session.id });
    } else {
      const id = `replay-${Date.now()}`;
      send({
        action: "enqueue",
        item: id,
        task_key: taskKey || "execute",
        prompt: `Replay: ${session.display_name}`,
        model: session.model,
        provider: session.provider,
      });
    }
  }

  return (
    <div className={`board-card${failed ? " failed" : ""}`}>
      <div className="card-top">
        {taskKey && <Badge type={taskKey} />}
      </div>
      {failed ? (
        <div className="card-failed-status">failed</div>
      ) : (
        <div className="card-done-status">done</div>
      )}
      <div className="card-name">{session.display_name}</div>
      <div className="card-footer">
        <WorktreeLabel name={session.worktree_name} />
        <span>{formatCost(session.total_cost_usd)}</span>
        <span className="model-tag">{session.model}</span>
        <button
          className="card-action-btn replay-btn"
          onClick={handleReplay}
          disabled={isPending}
          title={failed ? "Retry" : "Replay"}
        >
          <RotateCcw size={12} />
        </button>
      </div>
    </div>
  );
}

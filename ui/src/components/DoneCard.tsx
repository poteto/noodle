import type { Session } from "~/client";
import { useSendControl, formatCost } from "~/client";
import { Badge } from "./Badge";
import { WorktreeLabel } from "./WorktreeLabel";
import { RotateCcw } from "lucide-react";
import { Tooltip } from "./Tooltip";

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
    <div className={`bg-bg-1 border-2 border-border p-[18px] cursor-pointer shadow-card transition-[transform,box-shadow] duration-150 ease-out hover:-translate-x-0.5 hover:-translate-y-1 hover:shadow-card-hover${failed ? " border-l-[3px] border-l-nred" : ""}`}>
      <div className="flex items-center gap-1.5 mb-2">
        {taskKey ? <Badge type={taskKey} /> : null}
      </div>
      {failed ? (
        <div className="flex items-center gap-1.5 mb-1 font-mono text-xs text-nred font-semibold">failed</div>
      ) : (
        <div className="flex items-center gap-1.5 mb-1.5 font-mono text-xs text-ngreen font-medium">done</div>
      )}
      <div className="font-bold text-[1.0625rem] text-text-0 mb-1">{session.display_name}</div>
      <div className="flex items-center gap-1.5 font-mono text-xs text-text-2 mt-0.5">
        <WorktreeLabel name={session.worktree_name} />
        <span>{formatCost(session.total_cost_usd)}</span>
        <span className="px-1.5 py-px bg-bg-3 text-[0.6875rem] text-text-2 ml-auto">{session.model}</span>
        <Tooltip content={failed ? "Retry" : "Replay"}>
        <button
          className="flex items-center justify-center w-6 h-6 p-0 bg-transparent border-[1.5px] border-border-subtle text-text-2 cursor-pointer shrink-0 transition-all duration-[0.12s] hover:not-disabled:border-ngreen hover:not-disabled:text-ngreen hover:not-disabled:bg-ngreen-dim active:not-disabled:scale-90 disabled:opacity-30 disabled:cursor-not-allowed"
          onClick={handleReplay}
          disabled={isPending}
        >
          <RotateCcw size={12} />
        </button>
        </Tooltip>
      </div>
    </div>
  );
}

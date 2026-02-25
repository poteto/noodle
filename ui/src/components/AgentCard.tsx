import type { Session } from "~/client";
import { middleTruncate, formatDuration, formatCost, useSendControl } from "~/client";
import { WorktreeLabel } from "./WorktreeLabel";
import { Badge } from "./Badge";
import { Square } from "lucide-react";

export function AgentCard({
  session,
  onClick,
}: {
  session: Session;
  onClick?: () => void;
}) {
  const { mutate: send, isPending } = useSendControl();
  const taskKey = session.display_name.split("-")[0] ?? "";

  function handleStop(e: React.MouseEvent) {
    e.stopPropagation();
    send({ action: "stop", name: session.id });
  }

  return (
    <div className="board-card clickable" onClick={onClick}>
      <div className="card-top">
        {taskKey && <Badge type={taskKey} />}
        {session.remote_host && (
          <span className="card-remote" title={session.remote_host}>
            cloud
          </span>
        )}
      </div>

      <div className="card-name">{session.display_name}</div>
      <div className="card-task">
        {middleTruncate(session.current_action || "working...", 80)}
      </div>

      <div className="card-progress">
        <div className="card-progress-track">
          <div
            className="card-progress-fill"
            style={{ width: `${Math.round(session.context_window_usage_pct)}%` }}
          />
        </div>
        <div className="card-progress-label">
          ctx {Math.round(session.context_window_usage_pct)}%
        </div>
      </div>

      <div className="card-footer">
        <WorktreeLabel name={session.worktree_name} />
        <span>{formatDuration(session.duration_seconds)}</span>
        <span className="footer-sep">/</span>
        <span>{formatCost(session.total_cost_usd)}</span>
        {session.dispatch_warning && (
          <span
            className="dispatch-warning"
            title={session.dispatch_warning}
          >
            !
          </span>
        )}
        <span className="model-tag">{session.model}</span>
        <button
          className="card-action-btn stop-btn"
          onClick={handleStop}
          disabled={isPending}
          title="Stop and return to queue"
        >
          <Square size={12} />
        </button>
      </div>
    </div>
  );
}

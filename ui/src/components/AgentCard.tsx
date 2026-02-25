import type { Session } from "~/client";
import { middleTruncate, useSendControl } from "~/client";
import { WorktreeLabel } from "./WorktreeLabel";
import { Badge } from "./Badge";
import { Square } from "lucide-react";

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m < 60) return `${m}m${s > 0 ? ` ${s}s` : ""}`;
  const h = Math.floor(m / 60);
  return `${h}h ${m % 60}m`;
}

function formatCost(usd: number): string {
  if (usd < 0.01) return "<$0.01";
  return `$${usd.toFixed(2)}`;
}

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

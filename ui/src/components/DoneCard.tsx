import type { Session } from "~/client";
import { Badge } from "./Badge";

function formatCost(usd: number): string {
  if (usd < 0.01) return "<$0.01";
  return `$${usd.toFixed(2)}`;
}

export function DoneCard({ session }: { session: Session }) {
  const failed = session.status === "failed";
  const taskKey = session.display_name.split("-")[0] ?? "";

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
        <span>{formatCost(session.total_cost_usd)}</span>
        <span className="model-tag">{session.model}</span>
      </div>
    </div>
  );
}

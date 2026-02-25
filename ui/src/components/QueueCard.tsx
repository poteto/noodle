import type { QueueItem } from "~/client";
import { Badge } from "./Badge";

export function QueueCard({ item }: { item: QueueItem }) {
  return (
    <div className="board-card">
      <div className="card-top">
        {item.task_key && <Badge type={item.task_key} />}
      </div>
      <div className="card-name">{item.title || item.id}</div>
      {item.prompt && (
        <div className="card-task">
          {item.prompt.length > 120
            ? item.prompt.slice(0, 120) + "..."
            : item.prompt}
        </div>
      )}
      <div className="card-footer">
        <span className="model-tag">{item.model}</span>
      </div>
    </div>
  );
}

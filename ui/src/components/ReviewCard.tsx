import type { PendingReviewItem } from "~/client";
import { Badge } from "./Badge";
import { WorktreeLabel } from "./WorktreeLabel";
import { ReviewActions } from "./ReviewActions";

export function ReviewCard({ item }: { item: PendingReviewItem }) {
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
        <WorktreeLabel name={item.worktree_name} />
        {item.model && <span className="model-tag">{item.model}</span>}
      </div>
      <ReviewActions itemId={item.id} />
    </div>
  );
}

import type { PendingReviewItem, ControlCommand } from "~/client";
import { useSendControl } from "~/client";
import { Badge } from "./Badge";

export function ReviewCard({ item }: { item: PendingReviewItem }) {
  const { mutate: send, isPending } = useSendControl();

  function handleMerge() {
    const cmd: ControlCommand = { action: "merge", item: item.id };
    send(cmd);
  }

  function handleReject() {
    const cmd: ControlCommand = { action: "reject", item: item.id };
    send(cmd);
  }

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
      <div className="card-review-actions">
        <button
          className="merge-btn"
          onClick={handleMerge}
          disabled={isPending}
        >
          merge
        </button>
        <button
          className="reject-btn"
          onClick={handleReject}
          disabled={isPending}
        >
          reject
        </button>
      </div>
    </div>
  );
}

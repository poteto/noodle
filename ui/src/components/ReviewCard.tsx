import type { PendingReviewItem } from "~/client";
import { Badge } from "./Badge";
import { WorktreeLabel } from "./WorktreeLabel";
import { ReviewActions } from "./ReviewActions";

export function ReviewCard({ item }: { item: PendingReviewItem }) {
  return (
    <div className="group cursor-pointer">
    <div className="bg-bg-1 border-2 border-border p-[18px] shadow-card transition-[transform,box-shadow] duration-150 ease-out group-hover:-translate-x-0.5 group-hover:-translate-y-1 group-hover:shadow-card-hover">
      <div className="flex items-center gap-1.5 mb-2">
        {item.task_key && <Badge type={item.task_key} />}
      </div>
      <div className="font-bold text-[1.0625rem] text-text-0 mb-1">{item.title || item.id}</div>
      {item.prompt && (
        <div className="text-[0.8125rem] text-text-2 leading-[1.4] mb-2.5 whitespace-nowrap overflow-hidden text-ellipsis">
          {item.prompt.length > 120
            ? item.prompt.slice(0, 120) + "..."
            : item.prompt}
        </div>
      )}
      <div className="flex items-center gap-1.5 font-mono text-xs text-text-2 mt-0.5">
        <WorktreeLabel name={item.worktree_name} />
        {item.model && <span className="px-1.5 py-px bg-bg-3 text-[0.6875rem] text-text-2 ml-auto">{item.model}</span>}
      </div>
      <ReviewActions itemId={item.id} />
    </div>
    </div>
  );
}

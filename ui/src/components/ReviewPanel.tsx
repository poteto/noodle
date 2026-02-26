import type { PendingReviewItem } from "~/client";
import type { ReviewAction } from "./ReviewActions";
import { useReviewDiff } from "~/client";
import { Badge } from "./Badge";
import { WorktreeLabel } from "./WorktreeLabel";
import { DiffViewer } from "./DiffViewer";
import { ReviewActions } from "./ReviewActions";
import { SidePanel } from "./SidePanel";

export function ReviewPanel({
  item,
  onClose,
}: {
  item: PendingReviewItem;
  onClose: () => void;
}) {
  const { data, isLoading, error } = useReviewDiff(item.id);

  function handleAction(action: ReviewAction) {
    if (action === "merge" || action === "reject") {
      onClose();
    }
  }

  return (
    <SidePanel defaultWidth={800} onClose={onClose}>
      <div className="px-5 pt-[18px] pb-3 border-b-2 border-border shrink-0">
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center gap-2">
            {item.task_key && <Badge type={item.task_key} />}
            <span className="font-bold text-[1.125rem] text-text-0 whitespace-nowrap overflow-hidden text-ellipsis">
              {item.title || item.id}
            </span>
          </div>
          <button
            className="bg-transparent border-2 border-border py-0.5 px-[10px] font-mono text-[0.8125rem] font-bold cursor-pointer text-text-1 hover:bg-bg-hover active:translate-x-px active:translate-y-px active:shadow-btn-active"
            onClick={onClose}
          >
            x
          </button>
        </div>
        <div className="flex items-center gap-2 font-mono text-xs text-text-2">
          <WorktreeLabel name={item.worktree_name} />
          {item.model && (
            <span className="py-px px-[6px] bg-bg-3 text-[0.6875rem] text-text-2">
              {item.model}
            </span>
          )}
        </div>
      </div>

      <DiffViewer
        diff={data?.diff ?? ""}
        stat={data?.stat ?? ""}
        isLoading={isLoading}
        error={error?.message}
      />

      <div className="px-5 py-3 border-t-2 border-border shrink-0">
        <ReviewActions itemId={item.id} onAction={handleAction} />
      </div>
    </SidePanel>
  );
}

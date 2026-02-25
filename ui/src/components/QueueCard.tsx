import type { QueueItem } from "~/client";
import { Badge } from "./Badge";
import { GripVertical } from "lucide-react";

export function QueueCard({
  item,
  index,
  onDragStart,
  onDragOver,
  onDrop,
  onDragEnd,
  isDragOver,
}: {
  item: QueueItem;
  index?: number;
  onDragStart?: (e: React.DragEvent, index: number) => void;
  onDragOver?: (e: React.DragEvent, index: number) => void;
  onDrop?: (e: React.DragEvent, index: number) => void;
  onDragEnd?: () => void;
  isDragOver?: boolean;
}) {
  const isSchedule = item.task_key === "schedule";
  const classes = [
    "bg-bg-1 border-2 border-border p-[18px] shadow-card transition-[transform,box-shadow] duration-150 ease-out hover:-translate-x-0.5 hover:-translate-y-1 hover:shadow-card-hover cursor-grab active:cursor-grabbing",
    isDragOver && "border-t-[3px] border-t-nyellow pt-[15px]",
    isSchedule && "border-l-4 border-l-norange bg-norange-bg",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div
      className={classes}
      draggable
      onDragStart={(e) => onDragStart?.(e, index ?? 0)}
      onDragOver={(e) => {
        e.preventDefault();
        onDragOver?.(e, index ?? 0);
      }}
      onDrop={(e) => {
        e.preventDefault();
        onDrop?.(e, index ?? 0);
      }}
      onDragEnd={onDragEnd}
    >
      <div className="flex items-center gap-1.5 mb-2">
        <span className="flex items-center text-text-3 cursor-grab active:cursor-grabbing shrink-0"><GripVertical size={14} /></span>
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
      {item.rationale && isSchedule && (
        <div className="font-mono text-xs text-text-2 leading-[1.4] mb-2 italic">{item.rationale}</div>
      )}
      <div className="flex items-center gap-1.5 font-mono text-xs text-text-2 mt-0.5">
        <span className="px-1.5 py-px bg-bg-3 text-[0.6875rem] text-text-2 ml-auto">{item.model}</span>
      </div>
    </div>
  );
}

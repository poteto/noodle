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
  return (
    <div
      className={`board-card draggable${isDragOver ? " drag-over" : ""}`}
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
      <div className="card-top">
        <span className="drag-handle"><GripVertical size={14} /></span>
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

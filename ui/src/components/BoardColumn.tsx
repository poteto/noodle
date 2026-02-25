import type { ReactNode } from "react";

export function BoardColumn({
  title,
  count,
  children,
}: {
  title: string;
  count: number;
  children: ReactNode;
}) {
  return (
    <div className="board-col">
      <div className="col-header">
        <div className="col-header-inner">
          <span className="col-title">{title}</span>
          <span className="col-count">{count}</span>
        </div>
      </div>
      <div className="col-cards">{children}</div>
    </div>
  );
}

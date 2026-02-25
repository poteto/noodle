import type { ReactNode } from "react";

export function BoardColumn({
  title,
  count,
  children,
  footer,
}: {
  title: string;
  count: number;
  children: ReactNode;
  footer?: ReactNode;
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
      {footer && <div className="col-footer">{footer}</div>}
    </div>
  );
}

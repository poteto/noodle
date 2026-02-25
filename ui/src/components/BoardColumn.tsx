import type { ReactNode } from "react";

export function BoardColumn({
  title,
  count,
  children,
  footer,
  emptyText,
  headerExtra,
}: {
  title: string;
  count: number;
  children: ReactNode;
  footer?: ReactNode;
  emptyText?: string;
  headerExtra?: ReactNode;
}) {
  return (
    <div className="board-col">
      <div className="col-header">
        <div className="col-header-inner">
          <span className="col-title">{title}</span>
          <div className="col-header-right">
            {headerExtra}
            <span className="col-count">{count}</span>
          </div>
        </div>
      </div>
      <div className="col-cards">
        {count === 0 && emptyText && (
          <div className="col-empty">{emptyText}</div>
        )}
        {children}
      </div>
      {footer && <div className="col-footer">{footer}</div>}
    </div>
  );
}

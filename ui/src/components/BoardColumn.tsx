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
    <div className="flex-1 min-w-[280px] max-w-[380px] flex flex-col max-h-full">
      <div className="border-t-5 border-border bg-bg-2 shrink-0 pb-3">
        <div className="flex items-center justify-between pt-2.5">
          <span className="font-display font-800 text-[1.75rem] tracking-[-0.01em] text-text-0">{title}</span>
          <div className="flex items-center gap-2">
            {headerExtra}
            <span className="font-mono text-[0.8125rem] font-700 px-2.5 py-0.5 bg-border text-bg-0">{count}</span>
          </div>
        </div>
      </div>
      <div className="flex flex-col gap-2.5 pb-5 overflow-y-auto min-h-0">
        {count === 0 && emptyText && (
          <div className="text-text-3 font-mono text-[0.8125rem] text-center px-5 py-10">{emptyText}</div>
        )}
        {children}
      </div>
      {footer && <div className="shrink-0 pt-2">{footer}</div>}
    </div>
  );
}

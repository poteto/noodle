import type { ErrorComponentProps } from "@tanstack/react-router";

export function BoardError({ error, reset }: ErrorComponentProps) {
  return (
    <div className="flex flex-col h-screen bg-bg-0">
      <div className="flex items-center justify-between px-10 pt-7 pb-[22px] border-b-3 border-border bg-bg-0 shrink-0">
        <div className="flex items-center gap-6">
          <h1 className="font-display font-extrabold text-[3.5rem] text-text-0 tracking-[-0.02em] leading-[0.85]">noodle</h1>
        </div>
      </div>
      <div className="flex flex-1 overflow-x-auto overflow-y-hidden px-10 py-8 gap-6 bg-bg-2 min-h-0">
        <div className="p-10 font-mono">
          <p className="text-nred mb-3">
            {error.message}
          </p>
          <button className="px-5 py-2 font-mono text-sm font-bold bg-accent text-bg-0 border-2 border-border cursor-pointer shadow-btn transition-[transform,box-shadow] duration-[0.12s] hover:-translate-x-0.5 hover:-translate-y-0.5 hover:shadow-btn-hover active:translate-x-px active:translate-y-px active:shadow-btn-active" onClick={reset}>
            retry
          </button>
        </div>
      </div>
    </div>
  );
}

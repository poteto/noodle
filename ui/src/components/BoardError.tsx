import { useState } from "react";
import type { ErrorComponentProps } from "@tanstack/react-router";
import { RotateCcw } from "lucide-react";

export function BoardError({ error, reset }: ErrorComponentProps) {
  const [retrying, setRetrying] = useState(false);

  function handleRetry() {
    setRetrying(true);
    reset();
    // If reset doesn't unmount us, stop the spinner after a beat.
    setTimeout(() => setRetrying(false), 2000);
  }

  return (
    <div className="flex flex-col h-screen bg-bg-0">
      <div className="flex items-center justify-between px-10 pt-7 pb-[22px] border-b-3 border-border bg-bg-0 shrink-0">
        <div className="flex items-center gap-6">
          <h1 className="font-display font-extrabold text-[3.5rem] text-text-0 tracking-[-0.02em] leading-[0.85]">
            noodle
          </h1>
        </div>
      </div>

      <div className="flex flex-1 items-center justify-center bg-bg-2 min-h-0 p-10">
        <div className="bg-bg-1 border-2 border-border border-l-[6px] border-l-nred p-10 shadow-poster-md max-w-[480px] w-full animate-fade-in">
          <h2 className="font-display font-extrabold text-[1.75rem] text-text-0 tracking-[-0.01em] mb-1">
            Something broke
          </h2>
          <p className="text-text-2 text-[0.875rem] mb-6">
            The dashboard couldn&apos;t load. This usually means the noodle server is down or
            restarting.
          </p>

          <div className="bg-bg-2 border border-border-subtle px-4 py-3 mb-8 font-mono text-[0.8125rem] text-nred break-all">
            {error.message}
          </div>

          <div className="group">
            <button
              type="button"
              className="flex items-center gap-2 px-6 py-2.5 bg-accent text-bg-0 font-display text-[0.9375rem] font-bold tracking-[0.04em] border-2 border-border shadow-btn cursor-pointer transition-[transform,box-shadow] duration-[0.12s] group-hover:-translate-x-0.5 group-hover:-translate-y-0.5 group-hover:shadow-btn-hover active:translate-x-px active:translate-y-px active:shadow-btn-active disabled:opacity-50 disabled:cursor-not-allowed disabled:shadow-none disabled:translate-x-0 disabled:translate-y-0"
              onClick={handleRetry}
              disabled={retrying}
            >
              <RotateCcw size={14} className={retrying ? "animate-spin" : ""} />
              {retrying ? "retrying..." : "retry"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

import { useState } from "react";
import type { ErrorComponentProps } from "@tanstack/react-router";
import { RotateCcw } from "lucide-react";

export function AppError({ error, reset }: ErrorComponentProps) {
  const [retrying, setRetrying] = useState(false);

  function handleRetry() {
    setRetrying(true);
    reset();
    // If reset doesn't unmount us, stop the spinner after a beat.
    setTimeout(() => setRetrying(false), 2000);
  }

  return (
    <div className="flex flex-col h-screen bg-bg-depth">
      <div className="flex items-center justify-between px-10 pt-7 pb-6 border-b border-border-subtle bg-bg-depth shrink-0">
        <h1 className="font-display font-bold text-lg tracking-wider uppercase text-text-primary">
          NOODLE
        </h1>
      </div>

      <div className="flex flex-1 items-center justify-center bg-bg-depth min-h-0 p-10">
        <div className="bg-bg-surface border border-border-subtle border-l-4 border-l-red p-10 max-w-[480px] w-full animate-fade-in">
          <h2 className="font-display font-bold text-xl text-text-primary mb-1">Something broke</h2>
          <p className="text-neutral-500 text-sm mb-6">
            The dashboard couldn&apos;t load. This usually means the noodle server is down or
            restarting.
          </p>

          <div className="bg-bg-depth border border-border-subtle px-4 py-3 mb-8 font-mono text-sm text-red break-all">
            {error.message}
          </div>

          <button
            type="button"
            className="flex items-center gap-2 px-6 py-2.5 bg-accent text-bg-depth font-display text-sm font-bold tracking-wider uppercase border border-border-subtle cursor-pointer hover:brightness-110 active:brightness-90 disabled:opacity-50 disabled:cursor-not-allowed"
            onClick={handleRetry}
            disabled={retrying}
          >
            <RotateCcw size={14} className={retrying ? "animate-spin" : ""} />
            {retrying ? "retrying..." : "retry"}
          </button>
        </div>
      </div>
    </div>
  );
}

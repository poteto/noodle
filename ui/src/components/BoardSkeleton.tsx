import { SkeletonCard } from "./SkeletonCard";

export function BoardSkeleton() {
  return (
    <div className="flex flex-col h-screen bg-bg-0">
      <div className="flex items-center justify-between px-10 pt-7 pb-[22px] border-b-3 border-border bg-bg-0 shrink-0">
        <div className="flex items-center gap-6">
          <h1 className="font-display font-extrabold text-[3.5rem] text-text-0 tracking-[-0.02em] leading-[0.85]">noodle</h1>
        </div>
      </div>
      <div className="flex flex-1 overflow-x-auto overflow-y-hidden px-10 py-8 gap-6 bg-bg-2 min-h-0">
        <div className="flex-1 min-w-[280px] max-w-[380px] flex flex-col max-h-full">
          <div className="border-t-5 border-border bg-bg-2 shrink-0 pb-3">
            <div className="flex items-center justify-between pt-2.5">
              <span className="font-display font-extrabold text-[1.75rem] tracking-[-0.01em] text-text-0">Queued</span>
              <div className="flex items-center gap-2">
                <span className="font-mono text-[0.8125rem] font-bold px-2.5 py-0.5 bg-accent text-bg-0">0</span>
              </div>
            </div>
          </div>
          <div className="flex flex-col gap-2.5 pb-5 overflow-y-auto min-h-0">
            <SkeletonCard />
          </div>
        </div>
        <div className="flex-1 min-w-[280px] max-w-[380px] flex flex-col max-h-full">
          <div className="border-t-5 border-border bg-bg-2 shrink-0 pb-3">
            <div className="flex items-center justify-between pt-2.5">
              <span className="font-display font-extrabold text-[1.75rem] tracking-[-0.01em] text-text-0">Cooking</span>
              <div className="flex items-center gap-2">
                <span className="font-mono text-[0.8125rem] font-bold px-2.5 py-0.5 bg-accent text-bg-0">0</span>
              </div>
            </div>
          </div>
          <div className="flex flex-col gap-2.5 pb-5 overflow-y-auto min-h-0">
            <div className="text-text-3 font-mono text-[0.8125rem] text-center px-5 py-10">No active cooks</div>
          </div>
        </div>
        <div className="flex-1 min-w-[280px] max-w-[380px] flex flex-col max-h-full">
          <div className="border-t-5 border-border bg-bg-2 shrink-0 pb-3">
            <div className="flex items-center justify-between pt-2.5">
              <span className="font-display font-extrabold text-[1.75rem] tracking-[-0.01em] text-text-0">Review</span>
              <div className="flex items-center gap-2">
                <span className="font-mono text-[0.8125rem] font-bold px-2.5 py-0.5 bg-accent text-bg-0">0</span>
              </div>
            </div>
          </div>
          <div className="flex flex-col gap-2.5 pb-5 overflow-y-auto min-h-0">
            <div className="text-text-3 font-mono text-[0.8125rem] text-center px-5 py-10">Nothing to review</div>
          </div>
        </div>
        <div className="flex-1 min-w-[280px] max-w-[380px] flex flex-col max-h-full">
          <div className="border-t-5 border-border bg-bg-2 shrink-0 pb-3">
            <div className="flex items-center justify-between pt-2.5">
              <span className="font-display font-extrabold text-[1.75rem] tracking-[-0.01em] text-text-0">Done</span>
              <div className="flex items-center gap-2">
                <span className="font-mono text-[0.8125rem] font-bold px-2.5 py-0.5 bg-accent text-bg-0">0</span>
              </div>
            </div>
          </div>
          <div className="flex flex-col gap-2.5 pb-5 overflow-y-auto min-h-0">
            <div className="text-text-3 font-mono text-[0.8125rem] text-center px-5 py-10">No completed tasks</div>
          </div>
        </div>
      </div>
    </div>
  );
}

export function SkeletonCard() {
  return (
    <div className="bg-bg-1 border-2 border-border border-l-4 border-l-norange p-[18px] shadow-card transition-[transform,box-shadow] duration-150 ease-out cursor-default pointer-events-none">
      <div className="flex items-center gap-[6px] mb-2">
        <span className="block bg-bg-3 rounded-sm animate-skeleton w-[56px] h-[18px]" />
      </div>
      <div className="block bg-bg-3 rounded-sm animate-skeleton w-3/4 h-4 mb-2" />
      <div className="block bg-bg-3 rounded-sm animate-skeleton w-full h-3 mb-[6px]" />
      <div className="block bg-bg-3 rounded-sm animate-skeleton w-3/5 h-3 mb-[6px]" />
      <div className="flex items-center gap-[6px] font-mono text-xs text-text-2 mt-0.5">
        <span className="block bg-bg-3 rounded-sm animate-skeleton w-12 h-[14px]" />
      </div>
    </div>
  );
}

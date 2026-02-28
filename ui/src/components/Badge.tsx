const BADGE_CLASS: Record<string, string> = {
  execute: "bg-[#d6e4c8] text-[#3a5028]",
  review: "bg-[#f0e0a8] text-[#6a5010]",
  reflect: "bg-[#e8d0d8] text-[#6a3848]",
  schedule: "bg-[#f0d8b8] text-[#6a4018]",
  plan: "bg-[#c8d8e8] text-[#2a4060]",
};

export function Badge({ type }: { type: string }) {
  const cls = BADGE_CLASS[type] ?? "bg-neutral-800 text-neutral-400";
  return (
    <span className={`font-mono text-[0.6875rem] font-bold px-2 py-0.5 inline-block ${cls}`}>
      {type}
    </span>
  );
}

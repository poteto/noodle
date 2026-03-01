const BADGE_CLASS: Record<string, string> = {
  execute: "bg-[#d6e4c8] text-[#3a5028] hover:bg-[#c4d8b2]",
  review: "bg-[#f0e0a8] text-[#6a5010] hover:bg-[#e6d494]",
  reflect: "bg-[#e8d0d8] text-[#6a3848] hover:bg-[#dcc0cc]",
  schedule: "bg-[#f0d8b8] text-[#6a4018] hover:bg-[#e6cca8]",
  plan: "bg-[#c8d8e8] text-[#2a4060] hover:bg-[#b8ccdc]",
};

export function Badge({ type }: { type: string }) {
  const cls = BADGE_CLASS[type] ?? "bg-neutral-800 text-neutral-400";
  return (
    <span
      className={`font-mono text-[0.6875rem] font-bold px-2 py-0.5 inline-block animate-scale-in transition-colors duration-150 ${cls}`}
    >
      {type}
    </span>
  );
}

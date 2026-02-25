const BADGE_CLASS: Record<string, string> = {
  execute: "badge-execute",
  review: "badge-review",
  reflect: "badge-reflect",
  prioritize: "badge-prioritize",
  plan: "badge-plan",
};

export function Badge({ type }: { type: string }) {
  const cls = BADGE_CLASS[type] ?? "badge-default";
  return <span className={`badge ${cls}`}>{type}</span>;
}

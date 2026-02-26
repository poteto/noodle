export function StatBadge({ label, value }: { label: string; value: string | number }) {
  return (
    <span className="flex items-center gap-[5px] px-2.5 py-1 bg-accent text-bg-0 border border-border font-semibold">
      {value} {label}
    </span>
  );
}

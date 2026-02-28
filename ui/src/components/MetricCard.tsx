export function MetricCard({
  label,
  value,
  unit,
}: {
  label: string;
  value: string;
  unit?: string;
}) {
  return (
    <div className="flex flex-col gap-1 p-3 bg-bg-depth border border-border-subtle">
      <span className="text-xs uppercase tracking-wider text-neutral-500">
        {label}
      </span>
      <span className="text-lg font-mono text-text-primary">
        {value}
        {unit && (
          <span className="text-xs text-neutral-500 ml-1">{unit}</span>
        )}
      </span>
    </div>
  );
}

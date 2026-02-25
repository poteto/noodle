export function StatBadge({
  label,
  value,
}: {
  label: string;
  value: string | number;
}) {
  return (
    <span className="stat-item">
      {value} {label}
    </span>
  );
}

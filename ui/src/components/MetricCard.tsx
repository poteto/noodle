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
    <div className="metric-card">
      <div className="metric-label">{label}</div>
      <div className="metric-value">
        {value}
        {unit && (
          <span style={{ fontSize: 12, color: "var(--color-text-tertiary)", marginLeft: 4 }}>
            {unit}
          </span>
        )}
      </div>
    </div>
  );
}

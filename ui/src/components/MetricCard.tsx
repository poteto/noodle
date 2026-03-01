import { useRef, useEffect, useState } from "react";

export function MetricCard({
  label,
  value,
  unit,
}: {
  label: string;
  value: string;
  unit?: string;
}) {
  const prevRef = useRef(value);
  const [flash, setFlash] = useState(false);

  useEffect(() => {
    if (prevRef.current !== value) {
      prevRef.current = value;
      setFlash(true);
      const timer = setTimeout(() => setFlash(false), 600);
      return () => clearTimeout(timer);
    }
  }, [value]);

  return (
    <div className="metric-card">
      <div className="metric-label">{label}</div>
      <div className={`metric-value ${flash ? "metric-value-updated" : ""}`}>
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

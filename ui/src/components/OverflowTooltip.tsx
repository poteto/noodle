import { useCallback, useRef, useState } from "react";
import { createPortal } from "react-dom";

interface OverflowTooltipProps {
  text: string;
  className?: string;
}

export function OverflowTooltip({ text, className }: OverflowTooltipProps) {
  const spanRef = useRef<HTMLSpanElement>(null);
  const [tip, setTip] = useState<{ x: number; y: number } | null>(null);

  const handleEnter = useCallback(() => {
    const el = spanRef.current;
    if (!el || el.scrollWidth <= el.clientWidth) return;
    const rect = el.getBoundingClientRect();
    setTip({ x: rect.left, y: rect.top });
  }, []);

  const handleLeave = useCallback(() => setTip(null), []);

  return (
    <>
      <span
        ref={spanRef}
        className={className}
        onMouseEnter={handleEnter}
        onMouseLeave={handleLeave}
      >
        {text}
      </span>
      {tip &&
        createPortal(
          <div
            className="overflow-tooltip"
            style={{ left: tip.x, top: tip.y }}
          >
            {text}
          </div>,
          document.body,
        )}
    </>
  );
}

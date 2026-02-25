import { useState, useRef, useCallback, useEffect, type ReactNode } from "react";

const triggerStyle = { display: "inline-flex" } as const;

export function Tooltip({
  content,
  children,
}: {
  content: string;
  children: ReactNode;
}) {
  const [visible, setVisible] = useState(false);
  const [position, setPosition] = useState<{ top: number; left: number }>({
    top: 0,
    left: 0,
  });
  const triggerRef = useRef<HTMLSpanElement>(null);
  const timeoutRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    return () => clearTimeout(timeoutRef.current);
  }, []);

  const show = useCallback(() => {
    clearTimeout(timeoutRef.current);
    timeoutRef.current = setTimeout(() => {
      if (!triggerRef.current) return;
      const rect = triggerRef.current.getBoundingClientRect();
      setPosition({
        top: rect.top - 4,
        left: rect.left + rect.width / 2,
      });
      setVisible(true);
    }, 400);
  }, []);

  const hide = useCallback(() => {
    clearTimeout(timeoutRef.current);
    setVisible(false);
  }, []);

  return (
    <>
      <span
        ref={triggerRef}
        onMouseEnter={show}
        onMouseLeave={hide}
        onFocus={show}
        onBlur={hide}
        style={triggerStyle}
      >
        {children}
      </span>
      {visible && (
        <div
          className="tooltip"
          style={{ top: position.top, left: position.left }}
          role="tooltip"
        >
          {content}
        </div>
      )}
    </>
  );
}

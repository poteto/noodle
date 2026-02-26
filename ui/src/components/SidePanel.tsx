import { useState, useEffect, useRef, useCallback } from "react";

interface SidePanelProps {
  defaultWidth: number;
  minWidth?: number;
  maxWidth?: number;
  onClose: () => void;
  children: React.ReactNode;
}

export function SidePanel({
  defaultWidth,
  minWidth = 400,
  maxWidth = 1200,
  onClose,
  children,
}: SidePanelProps) {
  const [width, setWidth] = useState(defaultWidth);
  const backdropRef = useRef<HTMLDivElement>(null);
  const dragging = useRef(false);

  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (e.key !== "Escape") return;
      const tag = (e.target as HTMLElement)?.tagName;
      if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;
      onClose();
    }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [onClose]);

  function handleBackdropClick(e: React.MouseEvent) {
    if (e.target === backdropRef.current) onClose();
  }

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      const newWidth = window.innerWidth - e.clientX;
      setWidth(Math.min(maxWidth, Math.max(minWidth, newWidth)));
    },
    [minWidth, maxWidth],
  );

  const handleMouseUp = useCallback(() => {
    dragging.current = false;
    document.body.classList.remove("select-none");
    document.removeEventListener("mousemove", handleMouseMove);
    document.removeEventListener("mouseup", handleMouseUp);
  }, [handleMouseMove]);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      dragging.current = true;
      document.body.classList.add("select-none");
      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
    },
    [handleMouseMove, handleMouseUp],
  );

  useEffect(() => {
    return () => {
      document.body.classList.remove("select-none");
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };
  }, [handleMouseMove, handleMouseUp]);

  return (
    <div
      className="fixed inset-0 bg-[rgba(26,20,0,0.3)] z-100 flex justify-end animate-fade-in"
      ref={backdropRef}
      onClick={handleBackdropClick}
    >
      <div
        className="h-screen bg-bg-1 border-l-[3px] border-border flex flex-col shadow-chat animate-slide-right relative"
        style={{ width, maxWidth: "100vw" }}
      >
        <div
          className="absolute top-0 left-0 bottom-0 w-1 cursor-col-resize bg-border hover:bg-accent z-10"
          onMouseDown={handleMouseDown}
        />
        {children}
      </div>
    </div>
  );
}

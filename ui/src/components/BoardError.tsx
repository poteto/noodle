import type { ErrorComponentProps } from "@tanstack/react-router";

export function BoardError({ error, reset }: ErrorComponentProps) {
  return (
    <div className="board-shell">
      <div className="board-header">
        <div className="board-header-left">
          <h1 className="board-title">noodle</h1>
        </div>
      </div>
      <div className="board-columns">
        <div style={{ padding: "40px", fontFamily: "var(--font-mono)" }}>
          <p style={{ color: "var(--red)", marginBottom: "12px" }}>
            {error.message}
          </p>
          <button className="loop-control-btn" onClick={reset}>
            retry
          </button>
        </div>
      </div>
    </div>
  );
}

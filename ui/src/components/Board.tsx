import { useSnapshot, deriveKanbanColumns } from "~/client";
import { BoardColumn } from "./BoardColumn";
import { AgentCard } from "./AgentCard";
import { QueueCard } from "./QueueCard";
import { ReviewCard } from "./ReviewCard";
import { DoneCard } from "./DoneCard";

export function Board() {
  const { data: snapshot, isLoading, error } = useSnapshot();

  if (isLoading || !snapshot) {
    return (
      <div className="board-shell">
        <div className="board-header">
          <div className="board-header-left">
            <h1 className="board-title">noodle</h1>
          </div>
        </div>
        <div className="board-columns">
          <p style={{ color: "var(--text-2)", fontFamily: "var(--font-mono)" }}>
            loading...
          </p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="board-shell">
        <div className="board-header">
          <div className="board-header-left">
            <h1 className="board-title">noodle</h1>
          </div>
        </div>
        <div className="board-columns">
          <p style={{ color: "var(--red)", fontFamily: "var(--font-mono)" }}>
            {error.message}
          </p>
        </div>
      </div>
    );
  }

  const columns = deriveKanbanColumns(snapshot);

  return (
    <div className="board-shell">
      <div className="board-header">
        <div className="board-header-left">
          <h1 className="board-title">noodle</h1>
          <div className="board-stats">
            <span className="loop-indicator">
              <span className="pulse-dot" />
              {snapshot.loop_state}
            </span>
            <span className="stat-item">
              {snapshot.active.length} cooking
            </span>
            <span className="stat-item">
              ${snapshot.total_cost_usd.toFixed(2)}
            </span>
          </div>
        </div>
      </div>

      <div className="board-columns">
        <BoardColumn title="Queued" count={columns.queued.length}>
          {columns.queued.map((item) => (
            <QueueCard key={item.id} item={item} />
          ))}
        </BoardColumn>

        <BoardColumn title="Cooking" count={columns.cooking.length}>
          {columns.cooking.map((session) => (
            <AgentCard key={session.id} session={session} />
          ))}
        </BoardColumn>

        <BoardColumn title="Review" count={columns.review.length}>
          {columns.review.map((item) => (
            <ReviewCard key={item.id} item={item} />
          ))}
        </BoardColumn>

        <BoardColumn title="Done" count={columns.done.length}>
          {columns.done.map((session) => (
            <DoneCard key={session.id} session={session} />
          ))}
        </BoardColumn>
      </div>
    </div>
  );
}

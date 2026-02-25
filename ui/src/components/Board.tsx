import { useState, useEffect, useCallback } from "react";
import { useSnapshot, deriveKanbanColumns, useSendControl } from "~/client";
import type { Session } from "~/client";
import { BoardHeader } from "./BoardHeader";
import { BoardColumn } from "./BoardColumn";
import { AgentCard } from "./AgentCard";
import { QueueCard } from "./QueueCard";
import { ReviewCard } from "./ReviewCard";
import { DoneCard } from "./DoneCard";
import { ChatPanel } from "./ChatPanel";
import { TaskEditor } from "./TaskEditor";

function isInputFocused(): boolean {
  const el = document.activeElement;
  if (!el) return false;
  const tag = el.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";
}

export function Board() {
  const { data: snapshot, isLoading, error } = useSnapshot();
  const { mutate: send } = useSendControl();
  const [selectedSession, setSelectedSession] = useState<Session | null>(null);
  const [showTaskEditor, setShowTaskEditor] = useState(false);

  const handleKeyboard = useCallback(
    (e: KeyboardEvent) => {
      if (isInputFocused()) return;
      if (e.key === "n") {
        e.preventDefault();
        setShowTaskEditor(true);
      }
      if (e.key === "p" && snapshot) {
        e.preventDefault();
        const isPaused = snapshot.loop_state === "paused";
        send({ action: isPaused ? "resume" : "pause" });
      }
    },
    [snapshot, send],
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyboard);
    return () => document.removeEventListener("keydown", handleKeyboard);
  }, [handleKeyboard]);

  if (error && !snapshot) {
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

  const columns = deriveKanbanColumns(snapshot);

  const liveSession = selectedSession
    ? snapshot.sessions.find((s) => s.id === selectedSession.id) ?? selectedSession
    : null;

  return (
    <div className="board-shell">
      <BoardHeader
        snapshot={snapshot}
        onNewTask={() => setShowTaskEditor(true)}
      />

      <div className="board-columns">
        <BoardColumn title="Queued" count={columns.queued.length}>
          {columns.queued.map((item) => (
            <QueueCard key={item.id} item={item} />
          ))}
        </BoardColumn>

        <BoardColumn title="Cooking" count={columns.cooking.length}>
          {columns.cooking.map((session) => (
            <AgentCard
              key={session.id}
              session={session}
              onClick={() => setSelectedSession(session)}
            />
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

      {liveSession && (
        <ChatPanel
          session={liveSession}
          onClose={() => setSelectedSession(null)}
        />
      )}

      {showTaskEditor && (
        <TaskEditor onClose={() => setShowTaskEditor(false)} />
      )}
    </div>
  );
}

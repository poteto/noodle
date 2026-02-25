import { useState, useEffect, useCallback, useRef } from "react";
import { useSuspenseSnapshot, deriveKanbanColumns, useSendControl } from "~/client";
import type { Session } from "~/client";
import { BoardHeader } from "./BoardHeader";
import { BoardColumn } from "./BoardColumn";
import { AgentCard } from "./AgentCard";
import { QueueCard } from "./QueueCard";
import { ReviewCard } from "./ReviewCard";
import { DoneCard } from "./DoneCard";
import { ChatPanel } from "./ChatPanel";
import { TaskEditor } from "./TaskEditor";
import { QueueAddCard } from "./QueueAddCard";
import { ConcurrencyBadge } from "./ConcurrencyBadge";
import { SkeletonCard } from "./SkeletonCard";

function isInputFocused(): boolean {
  const el = document.activeElement;
  if (!el) return false;
  const tag = el.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";
}

export function Board() {
  const { data: snapshot } = useSuspenseSnapshot();
  const { mutate: send } = useSendControl();
  const [selectedSession, setSelectedSession] = useState<Session | null>(null);
  const [showTaskEditor, setShowTaskEditor] = useState(false);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
  const [cookingDragOver, setCookingDragOver] = useState(false);
  const dragItemId = useRef<string | null>(null);

  const isPaused = snapshot.loop_state === "paused";

  const handleKeyboard = useCallback(
    (e: KeyboardEvent) => {
      if (isInputFocused()) return;
      if (e.key === "n") {
        e.preventDefault();
        setShowTaskEditor(true);
      }
      if (e.key === "p") {
        e.preventDefault();
        send({ action: isPaused ? "resume" : "pause" });
      }
    },
    [isPaused, send],
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyboard);
    return () => document.removeEventListener("keydown", handleKeyboard);
  }, [handleKeyboard]);

  const columns = deriveKanbanColumns(snapshot);
  const maxCooks = snapshot.max_cooks || 4;
  // Show skeleton when loop just started and scheduler hasn't produced items yet.
  const showQueueSkeleton =
    snapshot.loop_state === "running" &&
    columns.queued.length === 0 &&
    columns.cooking.length === 0 &&
    columns.done.length === 0;

  const liveSession = selectedSession
    ? snapshot.sessions.find((s) => s.id === selectedSession.id) ?? selectedSession
    : null;

  function handleQueueDragStart(e: React.DragEvent, index: number) {
    const item = columns.queued[index];
    if (!item) return;
    dragItemId.current = item.id;
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("text/plain", item.id);
  }

  function handleQueueDragOver(_e: React.DragEvent, index: number) {
    setDragOverIndex(index);
  }

  function handleQueueDrop(_e: React.DragEvent, dropIndex: number) {
    const id = dragItemId.current;
    if (!id) return;
    const srcIndex = columns.queued.findIndex((item) => item.id === id);
    if (srcIndex < 0 || srcIndex === dropIndex) return;
    const fullQueueIndex = snapshot.queue.findIndex(
      (item) => item.id === columns.queued[dropIndex]?.id,
    );
    if (fullQueueIndex >= 0) {
      send({ action: "reorder", item: id, value: String(fullQueueIndex) });
    }
    resetDrag();
  }

  function handleCookingDragOver(e: React.DragEvent) {
    if (dragItemId.current) {
      e.preventDefault();
      setCookingDragOver(true);
    }
  }

  function handleCookingDragLeave() {
    setCookingDragOver(false);
  }

  function handleCookingDrop(e: React.DragEvent) {
    e.preventDefault();
    const id = dragItemId.current;
    if (!id) return;
    send({ action: "reorder", item: id, value: "0" });
    if (columns.cooking.length >= maxCooks) {
      send({
        action: "set-max-cooks",
        value: String(columns.cooking.length + 1),
      });
    }
    resetDrag();
  }

  function resetDrag() {
    dragItemId.current = null;
    setDragOverIndex(null);
    setCookingDragOver(false);
  }

  return (
    <div className="board-shell">
      <BoardHeader
        snapshot={snapshot}
        onNewTask={() => setShowTaskEditor(true)}
      />

      <div className="board-columns">
        <BoardColumn
          title="Queued"
          count={columns.queued.length}
          footer={<QueueAddCard />}
          emptyText={showQueueSkeleton ? undefined : "No tasks queued"}
        >
          {showQueueSkeleton && <SkeletonCard />}
          {columns.queued.map((item, i) => (
            <QueueCard
              key={item.id}
              item={item}
              index={i}
              onDragStart={handleQueueDragStart}
              onDragOver={handleQueueDragOver}
              onDrop={handleQueueDrop}
              onDragEnd={resetDrag}
              isDragOver={dragOverIndex === i}
            />
          ))}
        </BoardColumn>

        <BoardColumn
          title="Cooking"
          count={columns.cooking.length}
          headerExtra={
            <ConcurrencyBadge
              active={columns.cooking.length}
              maxCooks={maxCooks}
            />
          }
        >
          <div
            className={`cooking-drop-zone${cookingDragOver ? " drag-over" : ""}`}
            onDragOver={handleCookingDragOver}
            onDragLeave={handleCookingDragLeave}
            onDrop={handleCookingDrop}
          >
            {columns.cooking.length === 0 && !cookingDragOver && (
              <div className="col-empty">No active cooks</div>
            )}
            {cookingDragOver && columns.cooking.length === 0 && (
              <div className="col-empty drop-hint">Drop to start cooking</div>
            )}
            {columns.cooking.map((session) => (
              <AgentCard
                key={session.id}
                session={session}
                onClick={() => setSelectedSession(session)}
              />
            ))}
          </div>
        </BoardColumn>

        <BoardColumn title="Review" count={columns.review.length} emptyText="Nothing to review">
          {columns.review.map((item) => (
            <ReviewCard key={item.id} item={item} />
          ))}
        </BoardColumn>

        <BoardColumn title="Done" count={columns.done.length} emptyText="No completed tasks">
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

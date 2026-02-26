import { useState, useEffect, useCallback, useRef, useOptimistic, useTransition } from "react";
import { useSuspenseSnapshot, deriveKanbanColumns, sendControl } from "~/client";
import type { Snapshot, Session, QueueItem, PendingReviewItem, ControlCommand } from "~/client";
import { BoardHeader } from "./BoardHeader";
import { BoardColumn } from "./BoardColumn";
import { AgentCard } from "./AgentCard";
import { QueueCard } from "./QueueCard";
import { ReviewCard } from "./ReviewCard";
import { DoneCard } from "./DoneCard";
import { ChatPanel } from "./ChatPanel";
import { ReviewPanel } from "./ReviewPanel";
import { TaskEditor } from "./TaskEditor";
import { QueueAddCard } from "./QueueAddCard";
import { ConcurrencyBadge } from "./ConcurrencyBadge";
import { SkeletonCard } from "./SkeletonCard";
import { ControlContext } from "./ControlContext";

type PanelState =
  | { type: "chat"; sessionId: string }
  | { type: "review"; item: PendingReviewItem }
  | null;

function pendingSession(item: QueueItem): Session {
  return {
    id: `pending-${item.id}`,
    display_name: item.title || item.id,
    title: item.title,
    task_key: item.task_key,
    status: "starting",
    runtime: "",
    provider: item.provider,
    model: item.model,
    total_cost_usd: 0,
    duration_seconds: 0,
    last_activity: new Date().toISOString(),
    current_action: "Starting...",
    health: "green",
    context_window_usage_pct: 0,
    retry_count: 0,
    idle_seconds: 0,
    stuck_threshold_seconds: 300,
    loop_state: "running",
  };
}

type OptimisticAction = { type: "move-to-cooking"; itemId: string } | ControlCommand;

function applyOptimisticSnapshot(current: Snapshot, action: OptimisticAction): Snapshot {
  // Special case: drag-to-cook creates a pending session placeholder.
  if ("type" in action) {
    const item = current.queue.find((q) => q.id === action.itemId);
    if (!item) {
      return current;
    }
    return {
      ...current,
      active_queue_ids: [...current.active_queue_ids, action.itemId],
      active: [...current.active, pendingSession(item)],
    };
  }

  switch (action.action) {
    case "pause": {
      return { ...current, loop_state: "paused" };
    }
    case "resume": {
      return { ...current, loop_state: "running" };
    }
    case "stop": {
      return {
        ...current,
        active: current.active.filter((s) => s.id !== action.name),
      };
    }
    case "merge":
    case "reject": {
      return {
        ...current,
        pending_reviews: current.pending_reviews.filter((r) => r.id !== action.item),
        pending_review_count: Math.max(0, current.pending_review_count - 1),
      };
    }
    case "request-changes": {
      // Don't optimistically remove — request-changes can no-op at max
      // concurrency, keeping the item in review. The next SSE snapshot
      // will reflect the real state.
      return current;
    }
    case "set-max-cooks": {
      const n = Number.parseInt(action.value ?? "", 10);
      return Number.isNaN(n) ? current : { ...current, max_cooks: n };
    }
    case "reorder": {
      if (!action.item || action.value === undefined) {
        return current;
      }
      const fromIndex = current.queue.findIndex((q) => q.id === action.item);
      const toIndex = Number.parseInt(action.value, 10);
      if (fromIndex === -1 || Number.isNaN(toIndex)) {
        return current;
      }
      const newQueue = [...current.queue];
      const [moved] = newQueue.splice(fromIndex, 1);
      newQueue.splice(toIndex, 0, moved);
      return { ...current, queue: newQueue };
    }
    case "requeue": {
      return {
        ...current,
        recent: current.recent.filter((s) => s.id !== action.item),
      };
    }
    default: {
      return current;
    }
  }
}

function isInputFocused(): boolean {
  const el = document.activeElement;
  if (!el) {
    return false;
  }
  const tag = el.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";
}

export function Board() {
  const { data: snapshot } = useSuspenseSnapshot();
  const [, startTransition] = useTransition();
  const [optimisticSnapshot, applyOptimistic] = useOptimistic(snapshot, applyOptimisticSnapshot);
  const [panelState, setPanelState] = useState<PanelState>(null);
  const [showTaskEditor, setShowTaskEditor] = useState(false);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
  const [draggingId, setDraggingId] = useState<string | null>(null);
  const [cookingDragOver, setCookingDragOver] = useState(false);
  const dragItemId = useRef<string | null>(null);

  const optimisticSend = useCallback(
    (command: ControlCommand) => {
      startTransition(async () => {
        applyOptimistic(command);
        await sendControl(command);
      });
    },
    [startTransition, applyOptimistic],
  );

  const isPaused = optimisticSnapshot.loop_state === "paused";

  const handleKeyboard = useCallback(
    (e: KeyboardEvent) => {
      if (isInputFocused()) {
        return;
      }
      if (e.key === "n") {
        e.preventDefault();
        setShowTaskEditor(true);
      }
      if (e.key === "p") {
        e.preventDefault();
        optimisticSend({ action: isPaused ? "resume" : "pause" });
      }
    },
    [isPaused, optimisticSend],
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyboard);
    return () => document.removeEventListener("keydown", handleKeyboard);
  }, [handleKeyboard]);

  const columns = deriveKanbanColumns(optimisticSnapshot);
  const maxCooks = optimisticSnapshot.max_cooks || 4;
  // Show skeleton when loop just started and scheduler hasn't produced items yet.
  const showQueueSkeleton =
    snapshot.loop_state === "running" &&
    columns.queued.length === 0 &&
    columns.cooking.length === 0 &&
    columns.done.length === 0;

  const selectedSession =
    panelState?.type === "chat"
      ? (snapshot.sessions.find((s) => s.id === panelState.sessionId) ?? null)
      : null;

  // Derive review panel visibility from fresh snapshot data — hides when item
  // leaves pending_reviews, and uses up-to-date metadata if item returns.
  const activeReviewItem =
    panelState?.type === "review"
      ? (optimisticSnapshot.pending_reviews.find((r) => r.id === panelState.item.id) ?? null)
      : null;

  function handleQueueDragStart(e: React.DragEvent, index: number) {
    const item = columns.queued[index];
    if (!item) {
      return;
    }
    dragItemId.current = item.id;
    setDraggingId(item.id);
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("text/plain", item.id);
  }

  function handleQueueDragOver(_e: React.DragEvent, index: number) {
    setDragOverIndex(index);
  }

  function handleQueueDrop(_e: React.DragEvent, dropIndex: number) {
    const id = dragItemId.current;
    if (!id) {
      return;
    }
    const srcIndex = columns.queued.findIndex((item) => item.id === id);
    if (srcIndex === -1 || srcIndex === dropIndex) {
      return;
    }
    const fullQueueIndex = snapshot.queue.findIndex(
      (item) => item.id === columns.queued[dropIndex]?.id,
    );
    if (fullQueueIndex !== -1) {
      optimisticSend({ action: "reorder", item: id, value: String(fullQueueIndex) });
    }
    resetDrag();
  }

  function handleCookingDragOver(e: React.DragEvent) {
    if (dragItemId.current && columns.cooking.length < maxCooks) {
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
    if (!id || columns.cooking.length >= maxCooks) {
      return;
    }
    startTransition(async () => {
      applyOptimistic({ type: "move-to-cooking", itemId: id });
      await sendControl({ action: "reorder", item: id, value: "0" });
    });
    resetDrag();
  }

  function resetDrag() {
    dragItemId.current = null;
    setDragOverIndex(null);
    setDraggingId(null);
    setCookingDragOver(false);
  }

  return (
    <ControlContext.Provider value={optimisticSend}>
      <div className="flex flex-col h-screen bg-bg-0">
        <BoardHeader snapshot={optimisticSnapshot} onNewTask={() => setShowTaskEditor(true)} />

        <div className="flex flex-1 overflow-x-auto overflow-y-hidden px-10 py-8 gap-6 bg-bg-2 min-h-0">
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
                isDragging={draggingId === item.id}
              />
            ))}
          </BoardColumn>

          <BoardColumn
            title="Cooking"
            count={columns.cooking.length}
            headerExtra={<ConcurrencyBadge active={columns.cooking.length} maxCooks={maxCooks} />}
          >
            <div
              className={`flex flex-col gap-2.5 min-h-[60px] transition-[background] duration-150${cookingDragOver ? " bg-nyellow-bg outline-2 outline-dashed outline-nyellow -outline-offset-2" : ""}`}
              onDragOver={handleCookingDragOver}
              onDragLeave={handleCookingDragLeave}
              onDrop={handleCookingDrop}
            >
              {columns.cooking.length === 0 && !cookingDragOver && (
                <div className="text-text-3 font-mono text-[0.8125rem] text-center px-5 py-10">
                  No active cooks
                </div>
              )}
              {cookingDragOver && columns.cooking.length === 0 && (
                <div className="text-nyellow font-mono text-[0.8125rem] text-center px-5 py-10 font-semibold">
                  Drop to start cooking
                </div>
              )}
              {columns.cooking.map((session) => (
                <AgentCard
                  key={session.id}
                  session={session}
                  onClick={() => setPanelState({ type: "chat", sessionId: session.id })}
                />
              ))}
            </div>
          </BoardColumn>

          <BoardColumn title="Review" count={columns.review.length} emptyText="Nothing to review">
            {columns.review.map((item) => (
              <ReviewCard
                key={item.id}
                item={item}
                onClick={() => setPanelState({ type: "review", item })}
              />
            ))}
          </BoardColumn>

          <BoardColumn title="Done" count={columns.done.length} emptyText="No completed tasks">
            {columns.done.map((session) => (
              <DoneCard
                key={session.id}
                session={session}
                onClick={() => setPanelState({ type: "chat", sessionId: session.id })}
              />
            ))}
          </BoardColumn>
        </div>

        {selectedSession && (
          <ChatPanel session={selectedSession} onClose={() => setPanelState(null)} />
        )}

        {activeReviewItem && (
          <ReviewPanel item={activeReviewItem} onClose={() => setPanelState(null)} />
        )}

        {showTaskEditor && <TaskEditor onClose={() => setShowTaskEditor(false)} />}
      </div>
    </ControlContext.Provider>
  );
}

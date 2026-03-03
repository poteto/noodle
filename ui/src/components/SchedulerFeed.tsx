import { useCallback, useMemo, useRef, useState } from "react";
import { useSuspenseSnapshot, useSessionEvents, useSendControl, formatCost } from "~/client";
import type { Session } from "~/client";
import { StreamingDelta } from "./StreamingDelta";
import { groupConsecutiveTools } from "./group-tools";
import { VirtualizedFeed } from "./VirtualizedFeed";

function findSchedulerSession(sessions: Session[]): Session | undefined {
  return sessions.find((s) => s.task_key?.toLowerCase().trim() === "schedule");
}

function isBootstrapScheduleSession(session: Session): boolean {
  return session.id.toLowerCase().startsWith("bootstrap-schedule");
}

function isScheduleOnlyOrder(order: { stages: { task_key?: string }[] }): boolean {
  return (
    order.stages.length > 0 &&
    order.stages.every((s) => s.task_key?.toLowerCase().trim() === "schedule")
  );
}

export function SchedulerFeed() {
  const { data: snapshot } = useSuspenseSnapshot();
  const { mutate: send, isPending } = useSendControl();
  const [input, setInput] = useState("");

  const schedulerSession = findSchedulerSession(snapshot.sessions);
  const activeSchedulerSession = snapshot.active.find(
    (s) => s.task_key?.toLowerCase().trim() === "schedule" && s.status === "running",
  );
  const bootstrapScheduleSession = snapshot.sessions.find(
    (s) => s.status === "running" && isBootstrapScheduleSession(s),
  );
  const isBootstrappingSchedule = Boolean(bootstrapScheduleSession);
  const isBootstrappingSchedulePending =
    snapshot.loop_state === "running" &&
    !schedulerSession &&
    snapshot.orders.some(isScheduleOnlyOrder);
  const isSchedulerThinking =
    snapshot.loop_state === "running" && Boolean(activeSchedulerSession?.current_action?.trim());
  const initialEvents = schedulerSession?.id
    ? snapshot.events_by_session[schedulerSession.id]
    : undefined;
  const { data: events = [] } = useSessionEvents(schedulerSession?.id, initialEvents);

  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const resizeTextarea = useCallback(() => {
    const el = textareaRef.current;
    if (!el) {
      return;
    }
    el.style.height = "auto";
    el.style.height = `${el.scrollHeight}px`;
  }, []);

  const items = useMemo(() => groupConsecutiveTools(events), [events]);

  let emptyMessage: string | undefined;
  if (events.length === 0) {
    if (isBootstrappingSchedule || isBootstrappingSchedulePending) {
      emptyMessage = "Bootstrapping schedule skill. Creating scheduler instructions now.";
    } else {
      emptyMessage = schedulerSession
        ? "No events yet."
        : "No scheduler session found. Send a prompt to start.";
    }
  }

  function handleSubmit() {
    const prompt = input.trim();
    if (!prompt) {
      return;
    }
    send({ action: "steer", target: "schedule", prompt });
    setInput("");
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
    }
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  }

  function handleStopAll() {
    send({ action: "stop-all" });
  }

  function handleStop() {
    if (schedulerSession) {
      send({ action: "stop", name: schedulerSession.id });
    }
  }

  function renderInputLabel() {
    if (isBootstrappingSchedule || isBootstrappingSchedulePending) {
      return "Bootstrapping schedule skill...";
    }
    if (isSchedulerThinking) {
      return (
        <>
          <span className="thinking-dots">
            <span className="thinking-dot" />
            <span className="thinking-dot" />
            <span className="thinking-dot" />
          </span>
          Thinking…
        </>
      );
    }
    return "Talk to the scheduler...";
  }

  return (
    <>
      <header className="feed-header">
        <div className="feed-title">
          Scheduler
          <span className="feed-badge badge-task">{snapshot.loop_state}</span>
          {(isBootstrappingSchedule || isBootstrappingSchedulePending) && (
            <span className="feed-badge badge-task">bootstrap</span>
          )}
          {schedulerSession && <span className="feed-badge">{schedulerSession.model}</span>}
        </div>
        <div className="feed-actions">
          <span
            style={{
              fontFamily: "var(--font-mono)",
              fontSize: 12,
              color: "var(--color-text-tertiary)",
            }}
          >
            {formatCost(snapshot.total_cost_usd)}
          </span>
          <button
            type="button"
            className="feed-action-btn stop-btn"
            onClick={handleStopAll}
            disabled={isPending}
          >
            Stop All
          </button>
        </div>
      </header>

      <div className="feed-watermark">NOODLE</div>

      <VirtualizedFeed
        items={items}
        emptyMessage={emptyMessage}
        tail={
          isSchedulerThinking && schedulerSession?.id ? (
            <StreamingDelta sessionId={schedulerSession.id} />
          ) : undefined
        }
      />

      <div className="input-area">
        <div className="input-label">{renderInputLabel()}</div>
        <div className="input-row">
          <div className="input-row-field">
            <textarea
              ref={textareaRef}
              value={input}
              onChange={(e) => {
                setInput(e.target.value);
                resizeTextarea();
              }}
              onKeyDown={handleKeyDown}
              placeholder="Enter instructions or critique..."
              rows={1}
            />
            <div className="input-hint">
              <kbd>Enter</kbd>
              <span>send</span>
              <kbd>Shift+Enter</kbd>
              <span>newline</span>
            </div>
          </div>
          <div className="input-row-actions">
            {isSchedulerThinking && (
              <button type="button" className="btn-stop" onClick={handleStop} disabled={isPending}>
                {isPending ? "Stopping…" : "Stop"}
              </button>
            )}
            <button
              type="button"
              className="btn-submit"
              onClick={handleSubmit}
              disabled={isPending || !input.trim()}
            >
              Send
            </button>
          </div>
        </div>
      </div>
    </>
  );
}

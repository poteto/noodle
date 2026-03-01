import { useLayoutEffect, useMemo, useRef, useState } from "react";
import type { KeyboardEvent, ReactNode } from "react";
import { Link } from "@tanstack/react-router";
import {
  formatCost,
  formatDuration,
  useSendControl,
  useSessionEvents,
  useSuspenseSnapshot,
} from "~/client";
import type { EventLine, Order, Session, StageStatus } from "~/client";

type NavKey = "dashboard" | "live" | "tree" | "review";
type FeedTone = "user" | "manager" | "worker";
type StageState = "running" | "pending" | "done";

type FeedEntry =
  | {
      kind: "text";
      id: string;
      at: string;
      role: string;
      tone: FeedTone;
      text: string;
    }
  | {
      kind: "read";
      id: string;
      at: string;
      role: string;
      tone: FeedTone;
      path: string;
      text: string;
      detail: string;
    };

interface StageItem {
  id: string;
  label: string;
  state: StageState;
}

interface OrderItem {
  id: string;
  title: string;
  active: boolean;
  stages: StageItem[];
}

const FALLBACK_ENTRIES: FeedEntry[] = [
  {
    kind: "text",
    id: "fallback-user",
    at: "08:52:11",
    role: "USER",
    tone: "user",
    text: "Compile a summary of recent topological data analysis developments as a technical brief.",
  },
  {
    kind: "text",
    id: "fallback-manager",
    at: "08:52:12",
    role: "SYS.MANAGER",
    tone: "manager",
    text: "Request acknowledged. Scheduler is delegating retrieval and synthesis to available agents.",
  },
];

const FALLBACK_ORDERS: OrderItem[] = [
  {
    id: "fallback-scheduling",
    title: "Scheduling Tasks",
    active: false,
    stages: [{ id: "fallback-schedule", label: "SCHEDULE", state: "running" }],
  },
  {
    id: "fallback-tda",
    title: "Generate TDA Summary",
    active: true,
    stages: [
      { id: "fallback-tda-execute", label: "EXECUTE", state: "running" },
      { id: "fallback-tda-quality", label: "QUALITY", state: "pending" },
      { id: "fallback-tda-reflect", label: "REFLECT", state: "pending" },
    ],
  },
];

function nowHMS(): string {
  return new Date().toTimeString().slice(0, 8);
}

function makeMessageID(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

function formatClock(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value.slice(11, 19) || value;
  }
  return parsed.toTimeString().slice(0, 8);
}

function findSchedulerSession(sessions: Session[]): Session | undefined {
  return sessions.find((session) => session.task_key?.toLowerCase().trim() === "schedule");
}

function toTone(label: string): FeedTone {
  const normalized = label.toLowerCase();
  if (normalized === "user") {
    return "user";
  }
  if (normalized === "manager") {
    return "manager";
  }
  return "worker";
}

function toRole(label: string): string {
  return label ? label.toUpperCase() : "SYSTEM";
}

function toStageState(status: StageStatus): StageState {
  if (status === "active" || status === "merging") {
    return "running";
  }
  if (status === "completed") {
    return "done";
  }
  return "pending";
}

function stageSymbol(state: StageState): string {
  if (state === "running") {
    return "▶";
  }
  if (state === "done") {
    return "✓";
  }
  return "□";
}

function orderFromSnapshot(order: Order): OrderItem {
  const stages: StageItem[] = order.stages.map((stage, index) => ({
    id: `${order.id}:${stage.session_id || stage.task_key || stage.skill || "stage"}:${index}`,
    label: (stage.task_key || stage.skill || `Stage ${index + 1}`).toUpperCase(),
    state: toStageState(stage.status),
  }));

  return {
    id: order.id,
    title: order.title || order.id,
    active: stages.some((stage) => stage.state === "running"),
    stages,
  };
}

function feedFromEvent(event: EventLine): FeedEntry {
  const common = {
    id: `${event.at}:${event.label}:${event.body}`,
    at: formatClock(event.at),
    role: toRole(event.label),
    tone: toTone(event.label),
  } as const;

  if (event.label.toLowerCase() === "read") {
    const [pathLine, ...rest] = event.body.split("\n");
    const path = pathLine.trim() || "(path unavailable)";
    const detail = rest.join(" ").trim();
    return {
      ...common,
      kind: "read",
      path,
      text: detail || "Read completed.",
      detail: "Source captured from session event stream.",
    };
  }

  return {
    ...common,
    kind: "text",
    text: event.body || "(empty event body)",
  };
}

function roleClass(tone: FeedTone): string {
  if (tone === "manager") {
    return "orch-role-manager";
  }
  if (tone === "worker") {
    return "orch-role-worker";
  }
  return "orch-role-user";
}

function treeNodeClass(state: StageState): string {
  if (state === "running") {
    return "active";
  }
  if (state === "done") {
    return "done";
  }
  return "";
}

function DashboardIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <rect x="3" y="3" width="7" height="7" />
      <rect x="14" y="3" width="7" height="7" />
      <rect x="14" y="14" width="7" height="7" />
      <rect x="3" y="14" width="7" height="7" />
    </svg>
  );
}

function FeedIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
      <circle cx="12" cy="7" r="4" />
    </svg>
  );
}

function TreeIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <circle cx="12" cy="12" r="5" />
      <line x1="12" y1="1" x2="12" y2="3" />
      <line x1="12" y1="21" x2="12" y2="23" />
      <line x1="4.2" y1="4.2" x2="5.6" y2="5.6" />
      <line x1="18.4" y1="18.4" x2="19.8" y2="19.8" />
      <line x1="1" y1="12" x2="3" y2="12" />
      <line x1="21" y1="12" x2="23" y2="12" />
      <line x1="4.2" y1="19.8" x2="5.6" y2="18.4" />
      <line x1="18.4" y1="5.6" x2="19.8" y2="4.2" />
    </svg>
  );
}

function ReviewIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
    </svg>
  );
}

function NavLink({
  active,
  icon,
  label,
  to,
}: {
  active: boolean;
  icon: ReactNode;
  label: string;
  to?: "/" | "/dashboard" | "/tree" | "/review";
}) {
  if (!to) {
    return (
      <span className={`orch-nav-link ${active ? "active" : ""}`}>
        {icon}
        <span>{label}</span>
      </span>
    );
  }

  return (
    <Link to={to} className={`orch-nav-link ${active ? "active" : ""}`}>
      {icon}
      <span>{label}</span>
    </Link>
  );
}

function FeedRow({ entry }: { entry: FeedEntry }) {
  return (
    <div className="orch-feed-row">
      <div className="orch-feed-time">{entry.at}</div>
      <div className="orch-feed-main">
        <div className={`orch-feed-role ${roleClass(entry.tone)}`}>{entry.role}</div>
        {entry.kind === "text" ? (
          <p className="orch-feed-text">{entry.text}</p>
        ) : (
          <div className="orch-read-card">
            <div className="orch-read-path">
              <span className="orch-tag">READ</span>
              <code>{entry.path}</code>
            </div>
            <p className="orch-feed-text">{entry.text}</p>
            <p className="orch-feed-detail">{entry.detail}</p>
          </div>
        )}
      </div>
    </div>
  );
}

export function OrchestratorView() {
  const { data: snapshot } = useSuspenseSnapshot();
  const { mutate: send } = useSendControl();
  const schedulerSession = findSchedulerSession(snapshot.sessions);
  const { data: schedulerEvents = [] } = useSessionEvents(schedulerSession?.id);

  const [activeNav] = useState<NavKey>("live");
  const [draft, setDraft] = useState("");
  const [localEntries, setLocalEntries] = useState<FeedEntry[]>([]);
  const [expandedOrders, setExpandedOrders] = useState<Record<string, boolean>>({});
  const feedRef = useRef<HTMLDivElement>(null);

  const orderItems = useMemo(() => {
    if (snapshot.orders.length === 0) {
      return FALLBACK_ORDERS;
    }
    return snapshot.orders.map(orderFromSnapshot);
  }, [snapshot.orders]);

  const mergedExpandedOrders = useMemo(() => {
    const next: Record<string, boolean> = {};
    for (const [index, order] of orderItems.entries()) {
      next[order.id] = expandedOrders[order.id] ?? (index === 0 || order.active);
    }
    return next;
  }, [expandedOrders, orderItems]);

  const eventEntries = useMemo(() => {
    if (schedulerEvents.length === 0) {
      return FALLBACK_ENTRIES;
    }
    return schedulerEvents.map((event) => feedFromEvent(event));
  }, [schedulerEvents]);

  const combinedEntries = useMemo(
    () => [...eventEntries, ...localEntries],
    [eventEntries, localEntries],
  );

  const focusOrder =
    orderItems.find((order) => order.active) || orderItems[0] || FALLBACK_ORDERS[0];

  const focusSession = useMemo(() => {
    if (!focusOrder) {
      return schedulerSession;
    }

    for (const stage of focusOrder.stages) {
      const matching = snapshot.sessions.find(
        (session) =>
          session.task_key?.toUpperCase() === stage.label || session.display_name === stage.label,
      );
      if (matching) {
        return matching;
      }
    }

    return snapshot.active[0] || schedulerSession;
  }, [focusOrder, schedulerSession, snapshot.active, snapshot.sessions]);

  const isRunning = schedulerSession?.status === "running";
  const contextPct = Math.round(focusSession?.context_window_usage_pct ?? 0);
  const rightHeader = focusOrder?.id || "no_active_order";

  useLayoutEffect(() => {
    const node = feedRef.current;
    if (node) {
      node.scrollTop = node.scrollHeight;
    }
  }, [combinedEntries.length]);

  function toggleOrder(orderID: string) {
    setExpandedOrders((current) => ({
      ...current,
      [orderID]: !mergedExpandedOrders[orderID],
    }));
  }

  function submitDraft() {
    const prompt = draft.trim();
    if (!prompt) {
      return;
    }

    if (schedulerSession) {
      send({ action: "steer", target: "schedule", prompt });
    } else {
      const userEntry: FeedEntry = {
        kind: "text",
        id: makeMessageID("local-user"),
        at: nowHMS(),
        role: "USER",
        tone: "user",
        text: prompt,
      };
      const managerEntry: FeedEntry = {
        kind: "text",
        id: makeMessageID("local-manager"),
        at: nowHMS(),
        role: "SYS.MANAGER",
        tone: "manager",
        text: "No live scheduler session detected yet. Prompt queued for when scheduler is online.",
      };
      setLocalEntries((current) => [...current, userEntry, managerEntry]);
    }

    setDraft("");
  }

  function onInputKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      submitDraft();
    }
  }

  return (
    <div className="orch-shell">
      <aside className="orch-left">
        <header className="orch-left-header">
          <div className="orch-logo-wrap">
            <span className="orch-logo-x">X</span>
            <span className="orch-logo-word">NOODLE</span>
          </div>
          <div className="orch-status">
            <span className={`orch-status-dot ${isRunning ? "is-live" : "is-idle"}`} />
            <span>{isRunning ? "running" : "idle"}</span>
          </div>
        </header>

        <nav className="orch-nav">
          <NavLink
            active={activeNav === "dashboard"}
            icon={<DashboardIcon />}
            label="Dashboard"
            to="/dashboard"
          />
          <NavLink active={activeNav === "live"} icon={<FeedIcon />} label="Live Feed" />
          <NavLink active={activeNav === "tree"} icon={<TreeIcon />} label="Tree" to="/tree" />
          <NavLink
            active={activeNav === "review"}
            icon={<ReviewIcon />}
            label="Review"
            to="/review"
          />
        </nav>

        <section className="orch-section">
          <h3>Agents</h3>
          <div className="orch-manager-card">
            <div className="orch-avatar manager">M</div>
            <div className="orch-manager-copy">
              <strong>Manager</strong>
              <span>{isRunning ? "Monitoring tasks" : "Waiting for scheduler"}</span>
            </div>
            <span className="orch-manager-dot" />
          </div>
          {snapshot.active.slice(0, 2).map((session) => (
            <button key={session.id} type="button" className="orch-agent-row">
              <span className="orch-avatar">
                {(session.display_name || session.id).slice(0, 2)}
              </span>
              <span>{session.display_name || session.id}</span>
              <span
                className={`orch-status-dot ${session.status === "running" ? "is-live" : "is-idle"}`}
              />
            </button>
          ))}
        </section>

        <section className="orch-orders">
          <div className="orch-orders-head">
            <h3>Orders</h3>
            <span>{orderItems.length}</span>
          </div>
          <div className="orch-order-list">
            {orderItems.map((order) => (
              <div key={order.id}>
                <button
                  type="button"
                  className={`orch-order-row ${order.active ? "active" : ""}`}
                  onClick={() => toggleOrder(order.id)}
                >
                  <span className={`orch-chevron ${mergedExpandedOrders[order.id] ? "open" : ""}`}>
                    ▸
                  </span>
                  <span className="orch-order-title">{order.title}</span>
                  <span className={`orch-order-dot ${order.active ? "active" : ""}`} />
                </button>
                {mergedExpandedOrders[order.id] && (
                  <div className="orch-stage-list">
                    {order.stages.map((stage) => (
                      <div key={stage.id} className={`orch-stage-row ${stage.state}`}>
                        <span className="orch-stage-glyph">{stageSymbol(stage.state)}</span>
                        <span>{stage.label}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        </section>

        <footer className="orch-left-footer">
          <span>Session Cost</span>
          <strong>{formatCost(snapshot.total_cost_usd)}</strong>
        </footer>
      </aside>

      <main className="orch-main">
        <header className="orch-main-header">
          <div className="orch-main-meta">
            <h1>{focusOrder.title}</h1>
            <span className="orch-badge">
              {focusOrder.stages[0]?.label.toLowerCase() || "execute"}
            </span>
            <span className="orch-model-pill">
              <span>{focusSession?.model || "unknown-model"}</span>
              <em>{isRunning ? "RUNNING" : "IDLE"}</em>
            </span>
          </div>
          <div className="orch-main-actions">
            <span>{formatCost(snapshot.total_cost_usd)}</span>
            <button type="button" onClick={() => send({ action: "stop-all" })}>
              Stop All
            </button>
          </div>
        </header>

        <div className="orch-feed" ref={feedRef}>
          {combinedEntries.map((entry) => (
            <FeedRow key={entry.id} entry={entry} />
          ))}
        </div>

        <div className="orch-input-wrap">
          <label htmlFor="orch-input">Steer this agent...</label>
          <div className="orch-input-row">
            <input
              id="orch-input"
              value={draft}
              onChange={(event) => setDraft(event.target.value)}
              onKeyDown={onInputKeyDown}
              placeholder="Enter command or feedback..."
            />
            <button type="button" className="ghost" onClick={() => send({ action: "stop-all" })}>
              Stop
            </button>
            <button type="button" className="accent" onClick={submitDraft}>
              Send
            </button>
          </div>
        </div>
      </main>

      <aside className="orch-right">
        <header className="orch-right-header">{rightHeader}</header>

        <div className="orch-metrics-grid">
          <div>
            <small>Cost</small>
            <strong>{formatCost(focusSession?.total_cost_usd ?? snapshot.total_cost_usd)}</strong>
          </div>
          <div>
            <small>Duration</small>
            <strong>{formatDuration(focusSession?.duration_seconds ?? 0)}</strong>
          </div>
          <div>
            <small>Context</small>
            <strong>{contextPct}%</strong>
          </div>
          <div>
            <small>Model</small>
            <strong>{focusSession?.model || "unknown"}</strong>
          </div>
        </div>

        <div className="orch-context-meter">
          <div className="label-row">
            <span>{contextPct}% context used</span>
          </div>
          <div className="meter-track">
            <div className="meter-fill" style={{ width: `${Math.min(contextPct, 100)}%` }} />
          </div>
        </div>

        <div className="orch-pipeline">
          <h3>Pipeline</h3>
          {focusOrder.stages.map((stage, index) => (
            <div
              key={stage.id}
              className={`orch-pipeline-row ${stage.state === "running" ? "active" : ""}`}
            >
              <span className="dot" />
              <span>{stage.label.toLowerCase()}</span>
              {index === 0 && stage.state === "running" && <em>running</em>}
            </div>
          ))}
          <p>
            {focusOrder.stages.filter((stage) => stage.state === "done").length}/
            {focusOrder.stages.length} stages
          </p>
        </div>

        <div className="orch-tree-panel">
          <div className="head">
            <h3>Execution Tree</h3>
            <span>{isRunning ? "LIVE" : "IDLE"}</span>
          </div>
          <ul>
            <li className="root">ROOT {rightHeader}</li>
            {focusOrder.stages.map((stage, index) => (
              <li key={stage.id} className={treeNodeClass(stage.state)}>
                {index + 1}. {stage.label}
              </li>
            ))}
          </ul>
          <div className="orch-sys-stats">
            <span>ACTIVE {snapshot.active.length}</span>
            <span>ORDERS {snapshot.orders.length}</span>
          </div>
        </div>
      </aside>
    </div>
  );
}

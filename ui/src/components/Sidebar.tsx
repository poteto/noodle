import { useState } from "react";
import { useActiveChannel, useSuspenseSnapshot, useWSStatus, formatCost } from "~/client";
import type { ChannelId, StageStatus } from "~/client";
import { Link, useLocation, useNavigate } from "@tanstack/react-router";

const stageStatusIcon: Record<StageStatus, { symbol: string; cls: string }> = {
  active: { symbol: "▶", cls: "stage-active" },
  merging: { symbol: "▶", cls: "stage-active" },
  pending: { symbol: "□", cls: "stage-pending" },
  completed: { symbol: "✓", cls: "stage-done" },
  failed: { symbol: "✗", cls: "stage-done" },
  cancelled: { symbol: "□", cls: "stage-pending" },
};

function DashboardIcon() {
  return (
    <svg
      className="nav-icon"
      viewBox="0 0 18 18"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <rect x="1" y="1" width="6" height="6" rx="1" />
      <rect x="11" y="1" width="6" height="6" rx="1" />
      <rect x="1" y="11" width="6" height="6" rx="1" />
      <rect x="11" y="11" width="6" height="6" rx="1" />
    </svg>
  );
}

function FeedIcon() {
  return (
    <svg
      className="nav-icon"
      viewBox="0 0 18 18"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <circle cx="9" cy="5" r="3" />
      <path d="M2 16c0-3.3 3.1-6 7-6s7 2.7 7 6" />
    </svg>
  );
}

function TreeIcon() {
  return (
    <svg
      className="nav-icon"
      viewBox="0 0 18 18"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <circle cx="9" cy="9" r="2.5" />
      <path d="M9 1v2M9 15v2M1 9h2M15 9h2M3.3 3.3l1.4 1.4M13.3 13.3l1.4 1.4M3.3 14.7l1.4-1.4M13.3 4.7l1.4-1.4" />
    </svg>
  );
}

function ReviewIcon() {
  return (
    <svg
      className="nav-icon"
      viewBox="0 0 18 18"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M9 1L2 4v5c0 4.4 3 7.5 7 9 4-1.5 7-4.6 7-9V4L9 1z" />
      <path d="M6 9l2 2 4-4" />
    </svg>
  );
}

function OrchestratorIcon() {
  return (
    <svg
      className="nav-icon"
      viewBox="0 0 18 18"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M2 3h14v12H2z" />
      <path d="M6 7h6M6 10h4" />
      <path d="M3 15h12" />
    </svg>
  );
}

function ConnectionDot() {
  const status = useWSStatus();
  let cls = "idle";
  if (status === "connected") {
    cls = "active";
  } else if (status === "connecting") {
    cls = "thinking";
  }
  return <div className={`status-dot ${cls}`} />;
}

function NavLink({
  to,
  label,
  active,
  icon,
}: {
  to: string;
  label: string;
  active: boolean;
  icon: React.ReactNode;
}) {
  return (
    <Link to={to} className={`nav-item ${active ? "active" : ""}`}>
      {icon}
      {label}
    </Link>
  );
}

function channelEq(a: ChannelId, b: ChannelId): boolean {
  if (a.type !== b.type) {
    return false;
  }
  if (a.type === "agent" && b.type === "agent") {
    return a.sessionId === b.sessionId;
  }
  return true;
}

function statusLabel(status: ReturnType<typeof useWSStatus>): string {
  if (status === "connected") {
    return "running";
  }
  if (status === "connecting") {
    return "connecting";
  }
  return "offline";
}

function stageNodeKey(stage: {
  task_key?: string;
  skill?: string;
  session_id?: string;
  status: string;
}): string {
  return [stage.task_key || stage.skill || "stage", stage.session_id || "none", stage.status].join(
    ":",
  );
}

export function Sidebar() {
  const { data: snapshot } = useSuspenseSnapshot();
  const { activeChannel } = useActiveChannel();
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const [expandedOrders, setExpandedOrders] = useState<Set<string>>(() => {
    const initial = new Set<string>();
    for (const order of snapshot.orders) {
      if (order.stages.some((s) => s.status === "active" || s.status === "merging")) {
        initial.add(order.id);
      }
    }
    return initial;
  });

  function selectChannel(channel: ChannelId) {
    if (channel.type === "scheduler") {
      navigate({ to: "/" });
    } else {
      navigate({ to: "/actor/$id", params: { id: channel.sessionId } });
    }
  }

  function toggleOrder(orderId: string) {
    setExpandedOrders((prev) => {
      const next = new Set(prev);
      if (next.has(orderId)) {
        next.delete(orderId);
      } else {
        next.add(orderId);
      }
      return next;
    });
  }

  const schedulerChannel: ChannelId = { type: "scheduler" };
  const isSchedulerActive = channelEq(activeChannel, schedulerChannel);
  const isFeedRoute = pathname === "/" || pathname.startsWith("/actor/");
  const wsStatus = useWSStatus();
  const currentStatusLabel = statusLabel(wsStatus);
  const schedulerRunning = snapshot.sessions.some(
    (s) => s.task_key?.toLowerCase().trim() === "schedule" && s.status === "running",
  );

  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <div className="logo-mark" />
        <span>NOODLE</span>
        <span className="header-status">
          <ConnectionDot /> {currentStatusLabel}
        </span>
      </div>

      <nav className="sidebar-nav">
        <NavLink
          to="/dashboard"
          label="Dashboard"
          active={pathname === "/dashboard"}
          icon={<DashboardIcon />}
        />
        <NavLink to="/" label="Live Feed" active={isFeedRoute} icon={<FeedIcon />} />
        <NavLink to="/tree" label="Tree" active={pathname === "/tree"} icon={<TreeIcon />} />
        <NavLink
          to="/review"
          label={`Review${snapshot.pending_review_count ? ` (${snapshot.pending_review_count})` : ""}`}
          active={pathname === "/review"}
          icon={<ReviewIcon />}
        />
        <NavLink
          to="/orchestrator"
          label="Orchestrator"
          active={pathname === "/orchestrator"}
          icon={<OrchestratorIcon />}
        />
      </nav>

      <div className="section-label">Agents</div>
      <ul className="agent-list">
        <li>
          <button
            type="button"
            className={`agent-item manager ${isSchedulerActive ? "active" : ""}`}
            onClick={() => selectChannel(schedulerChannel)}
          >
            <div className="agent-avatar">M</div>
            <div className="agent-info">
              <span className="agent-name">Manager</span>
              <span className="agent-meta-line">
                {schedulerRunning ? "MONITORING THE SITUATION" : "IDLE"}
              </span>
            </div>
            <div className={`status-dot ${wsStatus === "connected" ? "active" : "idle"}`} />
          </button>
        </li>
      </ul>

      <div className="section-label">
        Orders <span className="section-count">{snapshot.orders.length}</span>
      </div>

      <div className="agent-tree">
        {snapshot.orders.length === 0 && (
          <div
            style={{
              padding: "6px 12px",
              color: "var(--color-text-tertiary)",
              fontFamily: "var(--font-mono)",
              fontSize: 11,
            }}
          >
            No orders
          </div>
        )}
        {snapshot.orders.map((order) => {
          const isExpanded = expandedOrders.has(order.id);
          const hasActiveStage = order.stages.some((s) => s.status === "active");
          const orderActive = order.stages.some((s) => {
            if (!s.session_id) {
              return false;
            }
            const ch: ChannelId = { type: "agent", sessionId: s.session_id };
            return channelEq(activeChannel, ch);
          });

          return (
            <div key={order.id}>
              <button
                type="button"
                className={`tree-order ${orderActive ? "active" : ""} ${hasActiveStage ? "has-active-stage" : ""}`}
                onClick={() => toggleOrder(order.id)}
              >
                <span className={`tree-chevron ${isExpanded ? "open" : ""}`}>▸</span>
                <span className="tree-label" title={order.title || order.id}>
                  {order.title || order.id}
                </span>
                {hasActiveStage && <div className="status-dot active" />}
              </button>
              <div className={`tree-stages ${isExpanded ? "open" : ""}`}>
                {order.stages.map((stage, i) => {
                  const agentChannel: ChannelId | null = stage.session_id
                    ? { type: "agent", sessionId: stage.session_id }
                    : null;
                  const info = stageStatusIcon[stage.status] || stageStatusIcon.pending;
                  const stageLabel = stage.task_key || stage.skill || `Stage ${i + 1}`;

                  return (
                    <button
                      type="button"
                      key={stageNodeKey(stage)}
                      className={`tree-stage ${info.cls} ${agentChannel ? "stage-clickable" : ""}`}
                      onClick={(event) => {
                        event.stopPropagation();
                        if (agentChannel) {
                          selectChannel(agentChannel);
                        }
                      }}
                      disabled={!agentChannel}
                      style={{ cursor: agentChannel ? "pointer" : "default" }}
                    >
                      <span className="tree-icon">{info.symbol}</span>
                      <span className="tree-stage-label" title={stageLabel}>
                        {stageLabel}
                      </span>
                    </button>
                  );
                })}
              </div>
            </div>
          );
        })}
      </div>

      <div className="sidebar-footer">
        <span className="footer-label">Session Cost</span>
        <span className="footer-value">{formatCost(snapshot.total_cost_usd)}</span>
      </div>
    </aside>
  );
}

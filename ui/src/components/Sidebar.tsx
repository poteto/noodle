import { useState } from "react";
import { useActiveChannel, useSuspenseSnapshot, useWSStatus, formatCost } from "~/client";
import type { ChannelId, StageStatus } from "~/client";
import { Link, useLocation, useNavigate } from "@tanstack/react-router";
import {
  LayoutGrid,
  User,
  Network,
  BadgeCheck,
  Play,
  Check,
  X,
  Square,
  ChevronRight,
} from "lucide-react";
import { OverflowTooltip } from "./OverflowTooltip";

const stageStatusIcon: Record<StageStatus, { icon: React.ReactNode; cls: string }> = {
  active: { icon: <Play size={10} />, cls: "stage-active" },
  merging: { icon: <Play size={10} />, cls: "stage-active" },
  pending: { icon: <Square size={10} />, cls: "stage-pending" },
  completed: { icon: <Check size={10} />, cls: "stage-done" },
  failed: { icon: <X size={10} />, cls: "stage-done" },
  cancelled: { icon: <Square size={10} />, cls: "stage-pending" },
};

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
  const visibleOrders = snapshot.orders.filter(
    (o) => !o.stages.every((s) => s.task_key?.toLowerCase().trim() === "schedule"),
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
          icon={<LayoutGrid className="nav-icon" size={16} strokeWidth={1.5} />}
        />
        <NavLink
          to="/"
          label="Live Feed"
          active={isFeedRoute}
          icon={<User className="nav-icon" size={16} strokeWidth={1.5} />}
        />
        <NavLink
          to="/topology"
          label="Topology"
          active={pathname === "/topology"}
          icon={<Network className="nav-icon" size={16} strokeWidth={1.5} />}
        />
        <NavLink
          to="/reviews"
          label={`Reviews${snapshot.pending_review_count ? ` (${snapshot.pending_review_count})` : ""}`}
          active={pathname === "/reviews"}
          icon={<BadgeCheck className="nav-icon" size={16} strokeWidth={1.5} />}
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
        Orders <span className="section-count">{visibleOrders.length}</span>
      </div>

      <div className="agent-tree">
        {visibleOrders.length === 0 && (
          <div className="orders-empty">Waiting for the scheduler</div>
        )}
        {visibleOrders.map((order) => {
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
                <ChevronRight className={`tree-chevron ${isExpanded ? "open" : ""}`} size={14} />
                <OverflowTooltip className="tree-label" text={order.title || order.id} />
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
                      <span className="tree-icon">{info.icon}</span>
                      <OverflowTooltip className="tree-stage-label" text={stageLabel} />
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

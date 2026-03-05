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
  active: { icon: <Play size={12} strokeWidth={2.25} />, cls: "stage-active" },
  merging: { icon: <Play size={12} strokeWidth={2.25} />, cls: "stage-active" },
  pending: { icon: <Square size={11} />, cls: "stage-pending" },
  completed: { icon: <Check size={11} />, cls: "stage-done" },
  failed: { icon: <X size={11} />, cls: "stage-done" },
  cancelled: { icon: <Square size={11} />, cls: "stage-pending" },
};

function ConnectionDot() {
  const status = useWSStatus();
  let cls = "idle";
  if (status === "connected") {
    cls = "active";
  } else if (status === "connecting") {
    cls = "thinking";
  }
  return <div className={`status-dot ${cls}`} role="status" aria-label={`Connection: ${status}`} />;
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
    <Link
      to={to}
      className={`nav-item ${active ? "active" : ""}`}
      aria-current={active ? "page" : undefined}
    >
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

function stageNodeKey(orderID: string, stageIndex: number): string {
  return `${orderID}:${stageIndex}`;
}

function isScheduleOnlyOrder(order: { stages: { task_key?: string }[] }): boolean {
  return (
    order.stages.length > 0 &&
    order.stages.every((s) => s.task_key?.toLowerCase().trim() === "schedule")
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
  const hasSchedulerSession = snapshot.sessions.some(
    (s) => s.task_key?.toLowerCase().trim() === "schedule",
  );
  const bootstrapScheduleSession = snapshot.sessions.find(
    (s) => s.id.toLowerCase().startsWith("bootstrap-schedule") && s.status === "running",
  );
  const bootstrapScheduleRunning = Boolean(bootstrapScheduleSession);
  const bootstrapSchedulePending =
    snapshot.loop_state === "running" &&
    !hasSchedulerSession &&
    snapshot.orders.some(isScheduleOnlyOrder);
  const visibleOrders = snapshot.orders.filter((o) => {
    if (!isScheduleOnlyOrder(o)) {
      return true;
    }
    return bootstrapScheduleRunning || bootstrapSchedulePending;
  });

  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <div className="logo-mark" />
        <span>NOODLE</span>
        <span className="header-status">
          <ConnectionDot /> {currentStatusLabel}
        </span>
      </div>

      <nav className="sidebar-nav" aria-label="Main navigation">
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
            <div className="agent-avatar">S</div>
            <div className="agent-info">
              <span className="agent-name">Scheduler</span>
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
          <div className="orders-empty">
            No orders yet. The scheduler will create orders once it reads your backlog.
          </div>
        )}
        {visibleOrders.map((order) => {
          const isExpanded = expandedOrders.has(order.id);
          const hasActiveStage = order.stages.some((s) => s.status === "active");
          const isBootstrapScheduleOrder =
            (bootstrapScheduleRunning || bootstrapSchedulePending) &&
            order.id.toLowerCase().trim() === "schedule" &&
            isScheduleOnlyOrder(order);
          const orderLabel = isBootstrapScheduleOrder
            ? "Bootstrapping schedule skill"
            : order.title || order.id;
          const orderActive = order.stages.some((s) => {
            let sessionID = s.session_id;
            if (
              !sessionID &&
              isBootstrapScheduleOrder &&
              s.task_key?.toLowerCase().trim() === "schedule" &&
              bootstrapScheduleSession
            ) {
              sessionID = bootstrapScheduleSession.id;
            }
            if (!sessionID) {
              return false;
            }
            const ch: ChannelId = { type: "agent", sessionId: sessionID };
            return channelEq(activeChannel, ch);
          });

          return (
            <div key={order.id}>
              <button
                type="button"
                className={`tree-order ${orderActive ? "active" : ""} ${hasActiveStage ? "has-active-stage" : ""}`}
                onClick={() => toggleOrder(order.id)}
                aria-expanded={isExpanded}
              >
                <ChevronRight className={`tree-chevron ${isExpanded ? "open" : ""}`} size={14} />
                <OverflowTooltip className="tree-label" text={orderLabel} />
                {hasActiveStage && <div className="status-dot active" />}
              </button>
              <div className={`tree-stages ${isExpanded ? "open" : ""}`}>
                {order.stages.map((stage, i) => {
                  let stageSessionID = stage.session_id;
                  if (
                    !stageSessionID &&
                    isBootstrapScheduleOrder &&
                    stage.task_key?.toLowerCase().trim() === "schedule" &&
                    bootstrapScheduleSession
                  ) {
                    stageSessionID = bootstrapScheduleSession.id;
                  }
                  const agentChannel: ChannelId | null = stageSessionID
                    ? { type: "agent", sessionId: stageSessionID }
                    : null;
                  const info = stageStatusIcon[stage.status] || stageStatusIcon.pending;
                  const stageBaseLabel = stage.task_key || stage.skill || "Stage";
                  const stageLabel = `${stageBaseLabel} ${i + 1}`;

                  return (
                    <button
                      type="button"
                      key={stageNodeKey(order.id, i)}
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

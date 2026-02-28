import { useActiveChannel, useSuspenseSnapshot, useSSEStatus, formatCost } from "~/client";
import type { ChannelId, StageStatus } from "~/client";
import { useRouter } from "@tanstack/react-router";

const stageIcon: Record<StageStatus, string> = {
  active: "\u25A0",
  pending: "\u25A1",
  completed: "\u2713",
  failed: "\u2717",
  cancelled: "\u25A1",
};

function NavLink({ href, label, active }: { href: string; label: string; active: boolean }) {
  return (
    <a
      href={href}
      className={`block px-3 py-1.5 text-xs uppercase tracking-wider font-body border-l-2 ${
        active
          ? "border-accent text-accent"
          : "border-transparent text-neutral-500 hover:text-text-primary hover:bg-white/5"
      }`}
    >
      {label}
    </a>
  );
}

function SSEDot() {
  const status = useSSEStatus();
  const color =
    status === "connected" ? "bg-green" : status === "connecting" ? "bg-yellow-400" : "bg-red";
  return <div className={`w-2 h-2 ${color}`} />;
}

function channelEq(a: ChannelId, b: ChannelId): boolean {
  if (a.type !== b.type) return false;
  if (a.type === "agent" && b.type === "agent") return a.sessionId === b.sessionId;
  return true;
}

export function Sidebar() {
  const { data: snapshot } = useSuspenseSnapshot();
  const { activeChannel, setActiveChannel } = useActiveChannel();
  const router = useRouter();
  const pathname = router.state.location.pathname;

  const schedulerChannel: ChannelId = { type: "scheduler" };
  const isSchedulerActive = channelEq(activeChannel, schedulerChannel);

  return (
    <aside className="flex flex-col border-r border-border-subtle bg-bg-surface h-full overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-border-subtle">
        <h1 className="text-lg font-display font-bold tracking-wider uppercase">NOODLE</h1>
        <SSEDot />
      </div>

      {/* Nav */}
      <nav className="py-2 border-b border-border-subtle">
        <NavLink href="/dashboard" label="DASHBOARD" active={pathname === "/dashboard"} />
        <NavLink href="/" label="LIVE FEED" active={pathname === "/"} />
        <NavLink href="/tree" label="TREE" active={pathname === "/tree"} />
      </nav>

      {/* Scrollable content */}
      <div className="flex-1 overflow-y-auto">
        {/* Scheduler section */}
        <div className="p-2">
          <div className="px-2 py-1.5 text-xs uppercase tracking-wider text-neutral-500">
            SCHEDULER
          </div>
          <button
            type="button"
            onClick={() => setActiveChannel(schedulerChannel)}
            className={`w-full text-left px-3 py-1.5 text-sm font-body flex items-center gap-2 cursor-pointer ${
              isSchedulerActive ? "text-accent bg-white/5" : "text-text-primary hover:bg-white/5"
            }`}
          >
            <span>Manager</span>
            <span className="ml-auto text-xs text-neutral-500 font-body">LLM</span>
          </button>
        </div>

        {/* Orders section */}
        <div className="p-2">
          <div className="px-2 py-1.5 text-xs uppercase tracking-wider text-neutral-500">
            ORDERS
          </div>
          {snapshot.orders.length === 0 && (
            <div className="px-3 py-1.5 text-xs text-neutral-600">No orders</div>
          )}
          {snapshot.orders.map((order) => (
            <div key={order.id} className="mb-1">
              <div className="px-3 py-1 text-sm text-text-primary truncate">
                {order.title || order.id}
              </div>
              {order.stages.map((stage, i) => {
                const agentChannel: ChannelId | null = stage.session_id
                  ? { type: "agent", sessionId: stage.session_id }
                  : null;
                const isActive = agentChannel ? channelEq(activeChannel, agentChannel) : false;
                return (
                  <button
                    key={stage.task_key || i}
                    type="button"
                    disabled={!agentChannel}
                    onClick={() => agentChannel && setActiveChannel(agentChannel)}
                    className={`w-full text-left pl-6 pr-3 py-0.5 text-xs font-body flex items-center gap-1.5 ${
                      isActive
                        ? "text-accent bg-white/5"
                        : agentChannel
                          ? "text-neutral-400 hover:bg-white/5 cursor-pointer"
                          : "text-neutral-600"
                    }`}
                  >
                    <span className={stage.status === "failed" ? "text-red" : stage.status === "active" ? "text-green" : ""}>
                      {stageIcon[stage.status] || "\u25A1"}
                    </span>
                    <span className="truncate">{stage.task_key || stage.skill || `Stage ${i + 1}`}</span>
                  </button>
                );
              })}
            </div>
          ))}
        </div>
      </div>

      {/* Footer */}
      <div className="p-3 border-t border-border-subtle text-xs text-neutral-500 font-body">
        {formatCost(snapshot.total_cost_usd)}
      </div>
    </aside>
  );
}

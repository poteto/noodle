import { useActiveChannel, useSuspenseSnapshot, useSessionEvents, formatCost, formatDuration } from "~/client";
import type { Snapshot, Session, Order, EventLine } from "~/client";
import { MetricCard } from "./MetricCard";
import { StageRail } from "./StageRail";

interface FileTouched {
  path: string;
  action: "read" | "edit" | "write";
}

function deriveFilesTouched(events: EventLine[]): FileTouched[] {
  const files: FileTouched[] = [];
  const seen = new Set<string>();
  for (const e of events) {
    if (["Read", "Edit", "Write"].includes(e.label)) {
      const path = e.body.split("\n")[0].trim();
      if (!path) continue;
      const key = `${e.label.toLowerCase()}:${path}`;
      if (!seen.has(key)) {
        seen.add(key);
        files.push({ path, action: e.label.toLowerCase() as "read" | "edit" | "write" });
      }
    }
  }
  return files;
}

function findOrderForSession(
  sessionId: string,
  snapshot: Snapshot,
): Order | undefined {
  return snapshot.orders.find((order) =>
    order.stages.some((stage) => stage.session_id === sessionId),
  );
}

const actionBadgeColor: Record<string, string> = {
  read: "text-neutral-400 bg-neutral-800",
  edit: "text-accent bg-accent/10",
  write: "text-green bg-green/10",
};

function contextWindowColor(pct: number): string {
  if (pct > 80) return "text-red";
  if (pct > 50) return "text-accent";
  return "text-green";
}

function SectionHeader({ children }: { children: React.ReactNode }) {
  return (
    <div className="text-sm font-display font-bold uppercase tracking-wider p-4 border-b border-border-subtle">
      {children}
    </div>
  );
}

function SchedulerContext({ snapshot }: { snapshot: Snapshot }) {
  const activeCount = snapshot.active.length;
  const orderCount = snapshot.orders.length;
  const warningCount = snapshot.warnings?.length ?? 0;

  return (
    <>
      <SectionHeader>System Status</SectionHeader>

      <div className="p-4 border-b border-border-subtle">
        <div className="flex items-center gap-2">
          <span className="text-xs uppercase tracking-wider text-neutral-500">Loop</span>
          <span className="text-xs font-mono text-accent uppercase">{snapshot.loop_state}</span>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2 p-4">
        <MetricCard label="Active" value={String(activeCount)} />
        <MetricCard label="Orders" value={String(orderCount)} />
        <MetricCard label="Cost" value={formatCost(snapshot.total_cost_usd)} />
        <MetricCard label="Warnings" value={String(warningCount)} />
      </div>

      {warningCount > 0 && (
        <div className="px-4 pb-4">
          <div className="text-xs uppercase tracking-wider text-neutral-500 mb-2">
            Warnings
          </div>
          <ul className="space-y-1">
            {snapshot.warnings.map((w, i) => (
              <li
                key={i}
                className="text-xs font-mono text-red px-2 py-1 bg-red/5 border border-red/20"
              >
                {w}
              </li>
            ))}
          </ul>
        </div>
      )}
    </>
  );
}

function AgentContext({
  session,
  snapshot,
}: {
  session: Session;
  snapshot: Snapshot;
}) {
  const { data: events } = useSessionEvents(session.id);
  const order = findOrderForSession(session.id, snapshot);
  const filesTouched = deriveFilesTouched(events ?? []);

  const completedStages = order
    ? order.stages.filter((s) => s.status === "completed").length
    : 0;
  const totalStages = order ? order.stages.length : 0;
  const progressPct = totalStages > 0 ? (completedStages / totalStages) * 100 : 0;

  const cwPct = Math.round(session.context_window_usage_pct);

  return (
    <>
      <SectionHeader>{session.display_name || session.id}</SectionHeader>

      <div className="grid grid-cols-2 gap-2 p-4">
        <MetricCard label="Cost" value={formatCost(session.total_cost_usd)} />
        <MetricCard
          label="Duration"
          value={formatDuration(session.duration_seconds)}
        />
        <MetricCard
          label="Context"
          value={`${cwPct}%`}
        />
        <MetricCard label="Model" value={session.model} />
      </div>

      {/* Context window bar */}
      <div className="px-4 pb-4">
        <div className="h-1.5 bg-neutral-800 w-full">
          <div
            className={`h-full progress-fill ${cwPct > 80 ? "bg-red" : cwPct > 50 ? "bg-accent" : "bg-green"}`}
            style={{ width: `${Math.min(cwPct, 100)}%` }}
          />
        </div>
        <div className={`text-xs mt-1 ${contextWindowColor(cwPct)}`}>
          {cwPct}% context used
        </div>
      </div>

      {/* Stage pipeline */}
      {order && (
        <div className="px-4 pb-4 border-t border-border-subtle pt-4">
          <div className="text-xs uppercase tracking-wider text-neutral-500 mb-3">
            Pipeline
          </div>
          <StageRail stages={order.stages} />

          {/* Progress bar */}
          {totalStages > 0 && (
            <div className="mt-3">
              <div className="h-1 bg-neutral-800 w-full">
                <div
                  className="h-full bg-green progress-fill"
                  style={{ width: `${progressPct}%` }}
                />
              </div>
              <div className="text-xs text-neutral-500 mt-1">
                {completedStages}/{totalStages} stages
              </div>
            </div>
          )}
        </div>
      )}

      {/* Files touched */}
      {filesTouched.length > 0 && (
        <div className="px-4 pb-4 border-t border-border-subtle pt-4">
          <div className="text-xs uppercase tracking-wider text-neutral-500 mb-2">
            Files ({filesTouched.length})
          </div>
          <ul className="space-y-1 max-h-48 overflow-y-auto">
            {filesTouched.map((f, i) => (
              <li
                key={i}
                className="flex items-center gap-2 text-xs font-mono"
              >
                <span
                  className={`px-1 py-0.5 text-[10px] uppercase font-bold ${actionBadgeColor[f.action]}`}
                >
                  {f.action}
                </span>
                <span className="text-neutral-400 truncate" title={f.path}>
                  {f.path}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </>
  );
}

export function ContextPanel() {
  const { activeChannel } = useActiveChannel();
  const { data: snapshot } = useSuspenseSnapshot();

  const session =
    activeChannel.type === "agent"
      ? snapshot.sessions.find((s) => s.id === activeChannel.sessionId)
      : undefined;

  return (
    <aside className="flex flex-col border-l border-border-subtle bg-bg-surface h-full overflow-hidden">
      <div className="flex-1 overflow-y-auto">
        {activeChannel.type === "scheduler" ? (
          <SchedulerContext snapshot={snapshot} />
        ) : session ? (
          <AgentContext session={session} snapshot={snapshot} />
        ) : (
          <div className="p-4 text-xs text-neutral-500">Session not found</div>
        )}
      </div>
    </aside>
  );
}

import {
  useActiveChannel,
  useSuspenseSnapshot,
  useReviewDiff,
  formatCost,
  formatDuration,
} from "~/client";
import type { Snapshot, Session, Order, PendingReviewItem } from "~/client";
import { MetricCard } from "./MetricCard";
import { StageRail } from "./StageRail";
import { DiffViewer } from "./DiffViewer";

function keyByOccurrence(values: string[]): { key: string; value: string }[] {
  const counts = new Map<string, number>();
  return values.map((value) => {
    const next = (counts.get(value) ?? 0) + 1;
    counts.set(value, next);
    return { key: `${value}:${next}`, value };
  });
}

function contextFillColor(cwPct: number): string {
  if (cwPct > 80) {
    return "var(--color-red)";
  }
  if (cwPct > 50) {
    return "var(--color-accent)";
  }
  return "var(--color-green)";
}

function findOrderForSession(sessionId: string, snapshot: Snapshot): Order | undefined {
  return snapshot.orders.find((order) =>
    order.stages.some((stage) => stage.session_id === sessionId),
  );
}

function SystemFooter({ snapshot }: { snapshot: Snapshot }) {
  return (
    <div className="context-footer">
      <div className="context-footer-grid">
        <div>
          <div className="context-footer-label">Loop</div>
          <div className="context-footer-value">{snapshot.loop_state}</div>
        </div>
        <div>
          <div className="context-footer-label">Active</div>
          <div className="context-footer-value">{snapshot.active.length}</div>
        </div>
        <div>
          <div className="context-footer-label">Orders</div>
          <div className="context-footer-value">{snapshot.orders.length}</div>
        </div>
        <div>
          <div className="context-footer-label">Cost</div>
          <div className="context-footer-value">{formatCost(snapshot.total_cost_usd)}</div>
        </div>
      </div>
    </div>
  );
}

function SchedulerContext({ snapshot }: { snapshot: Snapshot }) {
  const warningCount = snapshot.warnings?.length ?? 0;
  const warningsWithKeys = keyByOccurrence(snapshot.warnings ?? []);
  const completedOrders = snapshot.orders.filter((o) => o.status === "completed").length;

  return (
    <>
      <div className="context-header">System Status</div>

      <div className="metric-grid">
        <MetricCard label="Loop" value={snapshot.loop_state} />
        <MetricCard label="Active" value={String(snapshot.active.length)} />
        <MetricCard label="Orders" value={String(snapshot.orders.length)} />
        <MetricCard label="Cost" value={formatCost(snapshot.total_cost_usd)} />
      </div>

      {snapshot.active.length > 0 && (
        <>
          <div className="ctx-section-label">Active Agents</div>
          <div className="stage-rail">
            {snapshot.active.map((s) => {
              const isRunning = s.status === "running";
              return (
                <div key={s.id} className={`stage-item ${isRunning ? "current" : ""}`}>
                  <div className={`stage-dot ${isRunning ? "active" : "pending"}`} />
                  <span>{s.display_name || s.id}</span>
                </div>
              );
            })}
          </div>
        </>
      )}

      {snapshot.orders.length > 0 && (
        <>
          <div className="ctx-section-label">Pipeline</div>
          <div className="ctx-progress">
            <div className="ctx-progress-label">
              <span>
                {completedOrders}/{snapshot.orders.length} orders
              </span>
            </div>
            <div className="ctx-progress-bar">
              <div
                className="ctx-progress-fill"
                style={{
                  width: `${snapshot.orders.length > 0 ? (completedOrders / snapshot.orders.length) * 100 : 0}%`,
                  background: "var(--color-green)",
                }}
              />
            </div>
          </div>
        </>
      )}

      {warningCount > 0 && (
        <>
          <div className="ctx-section-label">Warnings</div>
          <div style={{ padding: "0 16px" }}>
            <div className="file-list">
              {warningsWithKeys.map((warning) => (
                <div
                  key={warning.key}
                  className="file-item"
                  style={{ color: "var(--color-red)", lineHeight: 1.8 }}
                >
                  {warning.value}
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </>
  );
}

function AgentContext({ session, snapshot }: { session: Session; snapshot: Snapshot }) {
  const order = findOrderForSession(session.id, snapshot);

  const completedStages = order ? order.stages.filter((s) => s.status === "completed").length : 0;
  const totalStages = order ? order.stages.length : 0;
  const progressPct = totalStages > 0 ? (completedStages / totalStages) * 100 : 0;
  const cwPct = Math.round(session.context_window_usage_pct);

  return (
    <>
      <div className="context-header">{session.display_name || session.id}</div>

      <div className="metric-grid">
        <MetricCard label="Cost" value={formatCost(session.total_cost_usd)} />
        <MetricCard label="Duration" value={formatDuration(session.duration_seconds)} />
        <MetricCard label="Context" value={`${cwPct}%`} />
        <MetricCard label="Model" value={session.model} />
      </div>

      {/* Context window bar */}
      <div className="ctx-progress">
        <div className="ctx-progress-bar">
          <div
            className="ctx-progress-fill"
            style={{
              width: `${Math.min(cwPct, 100)}%`,
              background: contextFillColor(cwPct),
            }}
          />
        </div>
        <div className="ctx-progress-label">
          <span>{cwPct}% context used</span>
        </div>
      </div>

      {/* Stage pipeline */}
      {order && (
        <>
          <div className="ctx-section-label">Pipeline</div>
          <StageRail stages={order.stages} />

          {totalStages > 0 && (
            <div className="ctx-progress" style={{ marginTop: 12 }}>
              <div className="ctx-progress-bar">
                <div
                  className="ctx-progress-fill"
                  style={{ width: `${progressPct}%`, background: "var(--color-green)" }}
                />
              </div>
              <div className="ctx-progress-label">
                <span>
                  {completedStages}/{totalStages} stages
                </span>
              </div>
            </div>
          )}
        </>
      )}
    </>
  );
}

function ReviewDiffPanel({ review }: { review: PendingReviewItem }) {
  const { data, isLoading, error } = useReviewDiff(review.order_id);

  return (
    <>
      <div className="context-header">Review Diff</div>
      <DiffViewer
        diff={data?.diff ?? ""}
        stat={data?.stat ?? ""}
        isLoading={isLoading}
        error={error?.message}
      />
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

  const pendingReview =
    activeChannel.type === "agent"
      ? snapshot.pending_reviews?.find((r) => r.session_id === activeChannel.sessionId)
      : undefined;

  let content = (
    <>
      <div className="context-header">Agent</div>
      <div className="context-body">
        <span
          style={{
            fontFamily: "var(--font-mono)",
            fontSize: 12,
            color: "var(--color-text-tertiary)",
          }}
        >
          Session not found
        </span>
      </div>
    </>
  );

  if (activeChannel.type === "scheduler") {
    content = <SchedulerContext snapshot={snapshot} />;
  } else if (pendingReview) {
    content = <ReviewDiffPanel review={pendingReview} />;
  } else if (session) {
    content = <AgentContext session={session} snapshot={snapshot} />;
  }

  return (
    <aside className="context-panel">
      <div style={{ flex: 1, overflowY: "auto" }}>{content}</div>
      <SystemFooter snapshot={snapshot} />
    </aside>
  );
}

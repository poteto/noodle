import type { Snapshot, Session, Order } from "~/client";

export interface TreeNodeData {
  name: string;
  type: "scheduler" | "order" | "stage";
  status: string;
  cost?: number;
  model?: string;
  currentAction?: string;
  sessionId?: string;
  children?: TreeNodeData[];
}

export function snapshotToHierarchy(snapshot: Snapshot): TreeNodeData {
  return {
    name: "Scheduler",
    type: "scheduler",
    status: snapshot.loop_state,
    children: snapshot.orders.map((order) => orderToNode(order, snapshot.sessions)),
  };
}

function orderToNode(order: Order, sessions: Session[]): TreeNodeData {
  return {
    name: order.title || order.id,
    type: "order",
    status: order.status,
    children: order.stages.map((stage) => {
      const session = stage.session_id
        ? sessions.find((s) => s.id === stage.session_id)
        : undefined;
      return {
        name: stage.task_key || "stage",
        type: "stage" as const,
        status: stage.status,
        cost: session?.total_cost_usd,
        model: session?.model,
        currentAction: session?.current_action,
        sessionId: session?.id,
      };
    }),
  };
}

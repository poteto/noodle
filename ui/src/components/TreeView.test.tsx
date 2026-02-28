import { describe, expect, it } from "vitest";
import { snapshotToHierarchy } from "./tree-utils";
import { buildSnapshot, buildOrder, buildSession, buildStage } from "~/test-utils";

describe("snapshotToHierarchy", () => {
  it("returns scheduler root with no children for empty snapshot", () => {
    const snapshot = buildSnapshot();
    const tree = snapshotToHierarchy(snapshot);

    expect(tree.name).toBe("Scheduler");
    expect(tree.type).toBe("scheduler");
    expect(tree.children).toEqual([]);
  });

  it("transforms snapshot with 2 orders into correct tree structure", () => {
    const session1 = buildSession({ id: "s1", model: "opus", total_cost_usd: 1.23 });
    const session2 = buildSession({ id: "s2", model: "sonnet", current_action: "Write tests" });
    const snapshot = buildSnapshot({
      orders: [
        buildOrder({
          id: "o1",
          title: "Fix auth",
          status: "active",
          stages: [
            buildStage({ task_key: "execute", status: "active", session_id: "s1" }),
            buildStage({ task_key: "quality", status: "pending" }),
          ],
        }),
        buildOrder({
          id: "o2",
          title: "Add logging",
          status: "active",
          stages: [
            buildStage({ task_key: "execute", status: "active", session_id: "s2" }),
          ],
        }),
      ],
      sessions: [session1, session2],
    });

    const tree = snapshotToHierarchy(snapshot);

    expect(tree.name).toBe("Scheduler");
    expect(tree.children).toHaveLength(2);

    const order1 = tree.children![0]!;
    expect(order1.name).toBe("Fix auth");
    expect(order1.type).toBe("order");
    expect(order1.status).toBe("active");
    expect(order1.children).toHaveLength(2);

    const stage1 = order1.children![0]!;
    expect(stage1.name).toBe("execute");
    expect(stage1.type).toBe("stage");
    expect(stage1.status).toBe("active");
    expect(stage1.cost).toBe(1.23);
    expect(stage1.model).toBe("opus");

    const stage2 = order1.children![1]!;
    expect(stage2.name).toBe("quality");
    expect(stage2.status).toBe("pending");
    expect(stage2.cost).toBeUndefined();

    const order2 = tree.children![1]!;
    expect(order2.name).toBe("Add logging");
    expect(order2.children).toHaveLength(1);
    expect(order2.children![0]!.currentAction).toBe("Write tests");
  });

  it("uses order id when title is missing", () => {
    const snapshot = buildSnapshot({
      orders: [buildOrder({ id: "o-abc", title: undefined })],
    });
    const tree = snapshotToHierarchy(snapshot);
    expect(tree.children![0]!.name).toBe("o-abc");
  });

  it("uses 'stage' as fallback name when task_key is missing", () => {
    const snapshot = buildSnapshot({
      orders: [
        buildOrder({
          stages: [buildStage({ task_key: undefined })],
        }),
      ],
    });
    const tree = snapshotToHierarchy(snapshot);
    expect(tree.children![0]!.children![0]!.name).toBe("stage");
  });

  it("attaches session data to stages with session_id", () => {
    const session = buildSession({
      id: "sess-1",
      total_cost_usd: 5.67,
      model: "haiku",
      current_action: "Reading file",
    });
    const snapshot = buildSnapshot({
      orders: [
        buildOrder({
          stages: [buildStage({ session_id: "sess-1", status: "active" })],
        }),
      ],
      sessions: [session],
    });

    const stage = snapshotToHierarchy(snapshot).children![0]!.children![0]!;
    expect(stage.cost).toBe(5.67);
    expect(stage.model).toBe("haiku");
    expect(stage.currentAction).toBe("Reading file");
    expect(stage.sessionId).toBe("sess-1");
  });
});

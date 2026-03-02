import { describe, expect, it } from "vitest";
import { buildSnapshot } from "../test-utils";
import { chooseNewerSnapshot } from "./snapshot-freshness";

describe("chooseNewerSnapshot", () => {
  it("returns incoming when there is no current snapshot", () => {
    const incoming = buildSnapshot({ updated_at: "2026-03-02T11:00:00Z" });
    expect(chooseNewerSnapshot(undefined, incoming)).toBe(incoming);
  });

  it("keeps current when incoming is older", () => {
    const current = buildSnapshot({ updated_at: "2026-03-02T11:00:00Z" });
    const incoming = buildSnapshot({ updated_at: "2026-03-02T10:59:59Z" });
    expect(chooseNewerSnapshot(current, incoming)).toBe(current);
  });

  it("uses incoming when it is newer", () => {
    const current = buildSnapshot({ updated_at: "2026-03-02T11:00:00Z" });
    const incoming = buildSnapshot({ updated_at: "2026-03-02T11:00:01Z" });
    expect(chooseNewerSnapshot(current, incoming)).toBe(incoming);
  });
});

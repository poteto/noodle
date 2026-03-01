import { useRef, useEffect, useCallback } from "react";
import { hierarchy, select, tree, zoom as createZoom, zoomIdentity } from "d3";
import type {
  D3ZoomEvent,
  HierarchyNode,
  HierarchyPointLink,
  HierarchyPointNode,
  ZoomBehavior,
  ZoomTransform,
} from "d3";
import { useSuspenseSnapshot, formatCost } from "~/client";
import { useNavigate } from "@tanstack/react-router";
import type { Snapshot } from "~/client";
import { snapshotToHierarchy } from "./tree-utils";
import type { TreeNodeData } from "./tree-utils";

const NODE_WIDTH = 140;
const NODE_HEIGHT = 60;
const NODE_HORIZONTAL_GAP = 10;
const NODE_VERTICAL_GAP = 40;

type PointNode = HierarchyPointNode<TreeNodeData>;

const statusDotColors: Record<string, string> = {
  active: "var(--color-accent)",
  running: "var(--color-green)",
  completed: "var(--color-green)",
  failed: "var(--color-red)",
  pending: "var(--color-border-subtle)",
  paused: "var(--color-accent)",
};

// Persist zoom transform across route navigations.
let savedTransform: ZoomTransform | null = null;

function esc(s: string): string {
  return s
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function nodeHTML(data: TreeNodeData): string {
  const isActive = data.status === "active" || data.status === "running";
  const borderColor = isActive ? "var(--color-accent)" : "var(--color-border-subtle)";
  const dotColor = statusDotColors[data.status] ?? "var(--color-border-subtle)";

  let detail = "";
  if (data.currentAction) {
    detail = `<div class="tree-node-detail">${esc(data.currentAction)}</div>`;
  } else if (data.model) {
    detail = `<div class="tree-node-detail">${esc(data.model)}</div>`;
  }

  const costLine =
    typeof data.cost === "number"
      ? `<div class="tree-node-detail" style="margin-top:2px">${formatCost(data.cost)}</div>`
      : "";

  return `<div xmlns="http://www.w3.org/1999/xhtml" class="tree-node-card" style="border-color:${borderColor}"><div class="tree-node-header"><span class="tree-node-dot" style="background:${dotColor}"></span><span class="tree-node-name">${esc(data.name)}</span></div>${detail}${costLine}</div>`;
}

function nodeKey(d: HierarchyNode<TreeNodeData>): string {
  return d
    .ancestors()
    .toReversed()
    .map((a) => a.data.name)
    .join("/");
}

function actorSessionId(data: TreeNodeData): string | null {
  if (data.type !== "stage" || !data.sessionId) {
    return null;
  }
  return data.sessionId;
}

function renderTree(
  svgEl: SVGSVGElement,
  snapshot: Snapshot,
  options: {
    zoomRef: React.MutableRefObject<ZoomBehavior<SVGSVGElement, unknown> | null>;
    onActorClick: (sessionId: string) => void;
  },
) {
  const { zoomRef, onActorClick } = options;
  const rootNode = hierarchy(snapshotToHierarchy(snapshot));
  const treeLayout = tree<TreeNodeData>().nodeSize([
    NODE_WIDTH + NODE_HORIZONTAL_GAP,
    NODE_HEIGHT + NODE_VERTICAL_GAP,
  ]);
  treeLayout(rootNode);

  const svg = select(svgEl);

  // Reuse or create the pan/zoom group.
  let g = svg.select<SVGGElement>("g.tree-content");
  if (g.empty()) {
    g = svg.append("g").attr("class", "tree-content");

    const width = svgEl.clientWidth || 800;
    const height = svgEl.clientHeight || 600;
    const zoomBehavior = createZoom<SVGSVGElement, unknown>()
      .scaleExtent([0.2, 3])
      .extent([
        [0, 0],
        [width, height],
      ])
      .on("zoom", (event: D3ZoomEvent<SVGSVGElement, unknown>) => {
        g.attr("transform", event.transform.toString());
        savedTransform = event.transform;
      });
    svg.call(zoomBehavior);
    zoomRef.current = zoomBehavior;

    if (savedTransform) {
      // Restore previous position.
      svg.call(zoomBehavior.transform, savedTransform);
    } else {
      // Center on the scheduler (root) node.
      const t = zoomIdentity.translate(width / 2, height / 2);
      svg.call(zoomBehavior.transform, t);
    }
  }

  // Edges — join by index (stateless paths, no identity needed).
  g.selectAll<SVGPathElement, HierarchyPointLink<TreeNodeData>>("path.link")
    .data(rootNode.links())
    .join("path")
    .attr("class", "link")
    .attr("d", (d) => {
      const s = d.source as PointNode;
      const t = d.target as PointNode;
      return `M${s.x},${s.y}C${s.x},${(s.y + t.y) / 2} ${t.x},${(s.y + t.y) / 2} ${t.x},${t.y}`;
    })
    .attr("fill", "none")
    .attr("stroke", (d) => {
      const { status } = (d.target as PointNode).data;
      return status === "active" || status === "running"
        ? "var(--color-accent)"
        : "var(--color-border-subtle)";
    })
    .attr("stroke-width", 1.5)
    .attr("stroke-dasharray", (d) =>
      (d.target as PointNode).data.status === "pending" ? "4 4" : "none",
    );

  // Nodes — join by path key so D3 reuses existing foreignObjects.
  g.selectAll<SVGForeignObjectElement, PointNode>("foreignObject.node")
    .data(rootNode.descendants(), (d) => nodeKey(d))
    .join(
      (enter) =>
        enter
          .append("foreignObject")
          .attr("class", "node")
          .attr("width", NODE_WIDTH)
          .attr("height", NODE_HEIGHT)
          .attr("overflow", "visible"),
      (update) => update,
      (exit) => exit.remove(),
    )
    .attr("x", (d) => (d.x ?? 0) - NODE_WIDTH / 2)
    .attr("y", (d) => (d.y ?? 0) - NODE_HEIGHT / 2)
    .classed("node-clickable", (d) => actorSessionId(d.data) !== null)
    .on("click", (_, d) => {
      const sessionId = actorSessionId(d.data);
      if (sessionId) {
        onActorClick(sessionId);
      }
    })
    .each(function renderNode(this: SVGForeignObjectElement, d) {
      this.innerHTML = nodeHTML(d.data);
    });
}

export function TreeView() {
  const { data: snapshot } = useSuspenseSnapshot();
  const navigate = useNavigate();
  const svgRef = useRef<SVGSVGElement>(null);
  const zoomRef = useRef<ZoomBehavior<SVGSVGElement, unknown> | null>(null);
  const handleActorClick = useCallback(
    (sessionId: string) => {
      navigate({ to: "/actor/$id", params: { id: sessionId } });
    },
    [navigate],
  );

  useEffect(() => {
    if (svgRef.current) {
      renderTree(svgRef.current, snapshot, { zoomRef, onActorClick: handleActorClick });
    }
  }, [snapshot, handleActorClick]);

  const handleZoomIn = useCallback(() => {
    if (svgRef.current && zoomRef.current) {
      select(svgRef.current).transition().duration(200).call(zoomRef.current.scaleBy, 1.4);
    }
  }, []);

  const handleZoomOut = useCallback(() => {
    if (svgRef.current && zoomRef.current) {
      select(svgRef.current)
        .transition()
        .duration(200)
        .call(zoomRef.current.scaleBy, 1 / 1.4);
    }
  }, []);

  return (
    <div
      className="h-full w-full relative"
      style={{
        background: "var(--color-bg-depth)",
        backgroundImage: "radial-gradient(circle, var(--color-border-subtle) 1px, transparent 1px)",
        backgroundSize: "24px 24px",
      }}
    >
      <svg ref={svgRef} className="w-full h-full" style={{ display: "block" }} />
      <div className="tree-zoom-controls">
        <button type="button" onClick={handleZoomIn}>
          +
        </button>
        <button type="button" onClick={handleZoomOut}>
          &minus;
        </button>
      </div>
    </div>
  );
}

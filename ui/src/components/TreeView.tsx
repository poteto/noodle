import { useRef, useEffect, useCallback } from "react";
import { Plus, Minus } from "lucide-react";
import { easeCubicOut, hierarchy, select, tree, zoom as createZoom, zoomIdentity } from "d3";
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

const NODE_WIDTH = 100;
const NODE_HEIGHT = 90;
const NODE_HORIZONTAL_GAP = 15;
const NODE_VERTICAL_GAP = 100;

type PointNode = HierarchyPointNode<TreeNodeData>;

const statusDotColors: Record<string, string> = {
  active: "var(--color-accent)",
  running: "var(--color-green)",
  completed: "var(--color-green)",
  failed: "var(--color-red)",
  pending: "#555",
  paused: "var(--color-accent)",
};

// Persist zoom transform across route navigations.
let savedTransform: ZoomTransform | null = null;

function clearTreeTooltips(): void {
  for (const el of document.querySelectorAll("[data-tree-tooltip]")) {
    el.remove();
  }
}

function esc(s: string): string {
  return s
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function nodeHTML(data: TreeNodeData): string {
  const isActive = data.status === "active" || data.status === "running";
  const borderColor = isActive ? "var(--color-accent)" : "#444";
  const dotColor = statusDotColors[data.status] ?? "var(--color-border-subtle)";

  const statusLine = `<div class="tree-node-status">${esc(data.status)}</div>`;

  let detail = "";
  if (data.currentAction) {
    detail = `<div class="tree-node-detail">${esc(data.currentAction)}</div>`;
  } else if (data.model) {
    detail = `<div class="tree-node-detail">${esc(data.model)}</div>`;
  }

  const costLine =
    typeof data.cost === "number"
      ? `<div class="tree-node-cost">${formatCost(data.cost)}</div>`
      : "";

  const cardClass = isActive ? "tree-node-card node-active" : "tree-node-card";
  const dotClass = isActive ? "tree-node-dot tree-node-dot-active" : "tree-node-dot";

  return `<div xmlns="http://www.w3.org/1999/xhtml" class="${cardClass}" style="border-color:${borderColor}"><div class="tree-node-header"><span class="${dotClass}" style="background:${dotColor}"></span><span class="tree-node-name">${esc(data.name)}</span></div>${statusLine}${detail}${costLine}</div>`;
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

function edgePath(d: HierarchyPointLink<TreeNodeData>): string {
  const s = d.source;
  const t = d.target;
  return `M${s.x},${s.y}C${s.x},${(s.y + t.y) / 2} ${t.x},${(s.y + t.y) / 2} ${t.x},${t.y}`;
}

function isEdgeActive(d: HierarchyPointLink<TreeNodeData>): boolean {
  const st = d.target.data.status;
  return st === "active" || st === "running";
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
  const treeLayout = tree<TreeNodeData>()
    .nodeSize([NODE_WIDTH + NODE_HORIZONTAL_GAP, NODE_HEIGHT + NODE_VERTICAL_GAP])
    .separation(() => 1);
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

  // Use separate groups so nodes always render above edges (SVG z-order).
  let edgeGroup = g.select<SVGGElement>("g.tree-edges");
  if (edgeGroup.empty()) {
    edgeGroup = g.append("g").attr("class", "tree-edges");
  }
  let nodeGroup = g.select<SVGGElement>("g.tree-nodes");
  if (nodeGroup.empty()) {
    nodeGroup = g.append("g").attr("class", "tree-nodes");
  }

  // Edges — enter/update/exit with transitions.
  const links = rootNode.links() as HierarchyPointLink<TreeNodeData>[];

  edgeGroup
    .selectAll<SVGPathElement, HierarchyPointLink<TreeNodeData>>("path.link")
    .data(links)
    .join(
      (enter) =>
        enter
          .append("path")
          .attr("class", "link tree-enter")
          .attr("fill", "none")
          .attr("stroke-width", 1.5)
          .attr("d", edgePath),
      (update) =>
        update.call((sel) => sel.transition().duration(300).ease(easeCubicOut).attr("d", edgePath)),
      (exit) => exit.transition().duration(200).style("opacity", 0).remove(),
    )
    .classed("edge-active", isEdgeActive)
    .attr("stroke", (d) => (isEdgeActive(d) ? "var(--color-accent)" : "#555"))
    .attr("stroke-width", 1.5)
    .attr("stroke-dasharray", (d) => (d.target.data.status === "pending" ? "4 4" : "none"));

  // Nodes — join by path key so D3 reuses existing foreignObjects.
  const nodes = nodeGroup
    .selectAll<SVGForeignObjectElement, PointNode>("foreignObject.node")
    .data(rootNode.descendants(), (d) => nodeKey(d))
    .join(
      (enter) =>
        enter
          .append("foreignObject")
          .attr("class", "node tree-enter")
          .attr("width", NODE_WIDTH)
          .attr("height", NODE_HEIGHT)
          .attr("overflow", "visible")
          .attr("x", (d) => (d.x ?? 0) - NODE_WIDTH / 2)
          .attr("y", (d) => (d.y ?? 0) - NODE_HEIGHT / 2),
      (update) => update,
      (exit) => exit.transition().duration(200).style("opacity", 0).remove(),
    );

  // Glide existing nodes to new positions.
  nodes
    .transition()
    .duration(300)
    .ease(easeCubicOut)
    .attr("x", (d) => (d.x ?? 0) - NODE_WIDTH / 2)
    .attr("y", (d) => (d.y ?? 0) - NODE_HEIGHT / 2);

  // Apply click handlers, tooltip, and render content on all nodes.
  nodes
    .classed("node-clickable", (d) => actorSessionId(d.data) !== null)
    .on("click", (_, d) => {
      const sessionId = actorSessionId(d.data);
      if (sessionId) {
        onActorClick(sessionId);
      }
    })
    .on("mouseenter", function onEnter(_event, d) {
      clearTreeTooltips();
      const rect = (this as SVGForeignObjectElement).getBoundingClientRect();
      const tip = document.createElement("div");
      tip.className = "overflow-tooltip";
      tip.textContent = d.data.name;
      tip.style.left = `${rect.left + rect.width / 2}px`;
      tip.style.top = `${rect.top}px`;
      tip.style.transform = "translateX(-50%) translateY(-100%) translateY(-6px)";
      tip.dataset.treeTooltip = "1";
      document.body.append(tip);
    })
    .on("mouseleave", clearTreeTooltips)
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
    return clearTreeTooltips;
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
          <Plus size={14} />
        </button>
        <button type="button" onClick={handleZoomOut}>
          <Minus size={14} />
        </button>
      </div>
    </div>
  );
}

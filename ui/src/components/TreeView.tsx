import { useRef, useEffect, useCallback } from "react";
import { createRoot } from "react-dom/client";
import * as d3 from "d3";
import { useSuspenseSnapshot } from "~/client";
import type { Snapshot } from "~/client";
import { snapshotToHierarchy } from "./tree-utils";
import type { TreeNodeData } from "./tree-utils";
import { TreeNodeCard } from "./TreeNode";

const NODE_WIDTH = 160;
const NODE_HEIGHT = 70;
const MARGIN = { top: 40, left: 40 };

type PointNode = d3.HierarchyPointNode<TreeNodeData>;

function renderTree(svgEl: SVGSVGElement, snapshot: Snapshot) {
  const root = d3.hierarchy(snapshotToHierarchy(snapshot));
  const treeLayout = d3.tree<TreeNodeData>().nodeSize([NODE_WIDTH + 40, NODE_HEIGHT + 60]);
  treeLayout(root);

  const svg = d3.select(svgEl);

  // Reuse or create the pan/zoom group.
  let g = svg.select<SVGGElement>("g.tree-content");
  if (g.empty()) {
    g = svg.append("g").attr("class", "tree-content");

    const zoom = d3
      .zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.2, 3])
      .on("zoom", (event: d3.D3ZoomEvent<SVGSVGElement, unknown>) => {
        g.attr("transform", event.transform.toString());
      });
    svg.call(zoom);

    const width = svgEl.clientWidth || 800;
    const t = d3.zoomIdentity.translate(width / 2 + MARGIN.left, MARGIN.top);
    svg.call(zoom.transform, t);
  }

  // Clear and redraw.
  g.selectAll("*").remove();

  // Edges.
  g.selectAll("path.link")
    .data(root.links())
    .join("path")
    .attr("class", "link")
    .attr("d", (d) => {
      const s = d.source as PointNode;
      const t = d.target as PointNode;
      return `M${s.x},${s.y}C${s.x},${(s.y + t.y) / 2} ${t.x},${(s.y + t.y) / 2} ${t.x},${t.y}`;
    })
    .attr("fill", "none")
    .attr("stroke", (d) => {
      const status = (d.target as PointNode).data.status;
      return status === "active" || status === "running"
        ? "var(--color-accent)"
        : "var(--color-border-subtle)";
    })
    .attr("stroke-width", 1.5)
    .attr("stroke-dasharray", (d) => {
      return (d.target as PointNode).data.status === "pending" ? "4 4" : "none";
    });

  // Nodes via foreignObject.
  const nodes = g
    .selectAll<SVGForeignObjectElement, PointNode>("foreignObject.node")
    .data(root.descendants())
    .join("foreignObject")
    .attr("class", "node")
    .attr("x", (d) => (d.x ?? 0) - NODE_WIDTH / 2)
    .attr("y", (d) => (d.y ?? 0) - NODE_HEIGHT / 2)
    .attr("width", NODE_WIDTH)
    .attr("height", NODE_HEIGHT)
    .attr("overflow", "visible");

  nodes.each(function (d) {
    const fo = d3.select(this);
    fo.selectAll("*").remove();
    const div = fo.append("xhtml:div").attr("class", "react-root");
    const el = div.node() as HTMLDivElement | null;
    if (el) {
      createRoot(el).render(<TreeNodeCard data={d.data} />);
    }
  });
}

export function TreeView() {
  const { data: snapshot } = useSuspenseSnapshot();
  const svgRef = useRef<SVGSVGElement>(null);

  const doRender = useCallback(() => {
    if (svgRef.current) {
      renderTree(svgRef.current, snapshot);
    }
  }, [snapshot]);

  useEffect(() => {
    doRender();
  }, [doRender]);

  return (
    <div
      className="h-screen w-full relative"
      style={{
        background: "var(--color-bg-depth)",
        backgroundImage:
          "radial-gradient(circle, var(--color-border-subtle) 1px, transparent 1px)",
        backgroundSize: "24px 24px",
      }}
    >
      <svg
        ref={svgRef}
        className="w-full h-full"
        style={{ display: "block" }}
      />
    </div>
  );
}

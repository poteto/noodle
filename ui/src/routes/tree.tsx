import { Suspense } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { TreeView } from "~/components/TreeView";

function TreePage() {
  return (
    <Suspense
      fallback={
        <div
          className="h-screen w-full flex items-center justify-center font-mono text-sm"
          style={{ background: "var(--color-bg-depth)", color: "var(--color-text-primary)" }}
        >
          Loading tree...
        </div>
      }
    >
      <TreeView />
    </Suspense>
  );
}

export const Route = createFileRoute("/tree")({
  component: TreePage,
});

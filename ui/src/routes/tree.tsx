import { createFileRoute } from "@tanstack/react-router";
import { TreeView } from "~/components/TreeView";

export const Route = createFileRoute("/tree")({
  component: TreeView,
});

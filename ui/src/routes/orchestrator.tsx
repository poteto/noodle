import { createFileRoute } from "@tanstack/react-router";
import { OrchestratorView } from "~/components/OrchestratorView";

export const Route = createFileRoute("/orchestrator")({
  component: OrchestratorView,
});

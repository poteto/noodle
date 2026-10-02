import { createFileRoute } from "@tanstack/react-router";
import { BacklogView } from "~/components/BacklogView";
import { AppError } from "~/components/AppError";

export const Route = createFileRoute("/backlog")({
  component: BacklogView,
  errorComponent: AppError,
});

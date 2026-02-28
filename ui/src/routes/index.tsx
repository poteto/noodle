import { createFileRoute } from "@tanstack/react-router";
import { AppLayout } from "~/components/AppLayout";
import { BoardError } from "~/components/BoardError";

export const Route = createFileRoute("/")({
  component: AppLayout,
  errorComponent: BoardError,
});

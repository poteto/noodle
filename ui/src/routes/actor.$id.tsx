import { createFileRoute } from "@tanstack/react-router";
import { AppLayout } from "~/components/AppLayout";
import { AppError } from "~/components/AppError";

export const Route = createFileRoute("/actor/$id")({
  component: AppLayout,
  errorComponent: AppError,
});

import { createFileRoute } from "@tanstack/react-router";
import { ReviewList } from "~/components/ReviewList";
import { AppError } from "~/components/AppError";

export const Route = createFileRoute("/review")({
  component: ReviewList,
  errorComponent: AppError,
});

import { Suspense } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { Board } from "~/components/Board";
import { BoardSkeleton } from "~/components/BoardSkeleton";
import { BoardError } from "~/components/BoardError";

function BoardPage() {
  return (
    <Suspense fallback={<BoardSkeleton />}>
      <Board />
    </Suspense>
  );
}

export const Route = createFileRoute("/")({
  component: BoardPage,
  errorComponent: BoardError,
});

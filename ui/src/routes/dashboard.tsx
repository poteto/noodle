import { Suspense } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { Dashboard } from "~/components/Dashboard";

function DashboardPage() {
  return (
    <Suspense fallback={<div className="h-screen bg-bg-depth" />}>
      <Dashboard />
    </Suspense>
  );
}

export const Route = createFileRoute("/dashboard")({
  component: DashboardPage,
});

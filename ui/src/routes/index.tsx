import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/")({
  component: () => (
    <div style={{ fontFamily: "system-ui", padding: 24 }}>
      <h1>noodle</h1>
      <p>Web UI loading...</p>
    </div>
  ),
});

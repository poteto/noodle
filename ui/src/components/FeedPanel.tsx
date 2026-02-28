import { useActiveChannel } from "~/client";
import { SchedulerFeed } from "./SchedulerFeed";
import { AgentFeed } from "./AgentFeed";

export function FeedPanel() {
  const { activeChannel } = useActiveChannel();

  return (
    <main className="flex flex-col h-full overflow-hidden bg-bg-depth">
      {activeChannel.type === "scheduler" ? (
        <SchedulerFeed />
      ) : (
        <AgentFeed sessionId={activeChannel.sessionId} />
      )}
    </main>
  );
}

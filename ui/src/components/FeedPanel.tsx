import { useActiveChannel } from "~/client";
import { SchedulerFeed } from "./SchedulerFeed";
import { AgentFeed } from "./AgentFeed";

function channelKey(ch: { type: string; sessionId?: string }): string {
  return ch.type === "agent" ? `agent-${ch.sessionId}` : "scheduler";
}

export function FeedPanel() {
  const { activeChannel } = useActiveChannel();

  return (
    <main
      key={channelKey(activeChannel)}
      className="flex flex-col h-full overflow-hidden bg-bg-depth animate-fade-in"
    >
      {activeChannel.type === "scheduler" ? (
        <SchedulerFeed />
      ) : (
        <AgentFeed sessionId={activeChannel.sessionId} />
      )}
    </main>
  );
}

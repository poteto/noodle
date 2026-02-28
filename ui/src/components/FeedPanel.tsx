import { useActiveChannel } from "~/client";
import { SchedulerFeed } from "./SchedulerFeed";

export function FeedPanel() {
  const { activeChannel } = useActiveChannel();

  return (
    <main className="flex flex-col h-full overflow-hidden bg-bg-depth">
      {activeChannel.type === "scheduler" ? (
        <SchedulerFeed />
      ) : (
        <div className="flex items-center justify-center h-full text-neutral-600 text-sm font-body">
          Agent feed coming soon
        </div>
      )}
    </main>
  );
}

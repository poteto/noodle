import { Sidebar } from "./Sidebar";
import { FeedPanel } from "./FeedPanel";
import { ContextPanel } from "./ContextPanel";

export function AppLayout() {
  return (
    <div className="grid grid-cols-[260px_1fr_300px] h-screen bg-bg-depth text-text-primary font-body">
      <Sidebar />
      <FeedPanel />
      <ContextPanel />
    </div>
  );
}

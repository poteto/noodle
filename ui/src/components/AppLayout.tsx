import { FeedKeyboardHandler } from "./FeedKeyboardHandler";
import { FeedPanel } from "./FeedPanel";
import { ContextPanel } from "./ContextPanel";

export function AppLayout() {
  return (
    <div className="grid grid-cols-[1fr_300px] h-full overflow-hidden">
      <FeedKeyboardHandler />
      <FeedPanel />
      <ContextPanel />
    </div>
  );
}

import type { EventLine } from "~/client";

const TOOL_BADGE: Record<string, string> = {
  Read: "bg-blue-900/50 text-blue-300",
  Edit: "bg-yellow-900/50 text-yellow-300",
  Bash: "bg-green-900/50 text-green-300",
  Write: "bg-purple-900/50 text-purple-300",
  Glob: "bg-blue-900/50 text-blue-300",
  Grep: "bg-blue-900/50 text-blue-300",
};

function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

function initials(label: string): string {
  return label.slice(0, 2).toUpperCase();
}

export function MessageRow({ event }: { event: EventLine }) {
  if (event.category === "ticket") {
    return (
      <div className="text-xs text-neutral-600 text-center py-1 font-body">
        {event.label}
      </div>
    );
  }

  const isThink = event.label === "Think";
  const isCost = event.label === "Cost";

  if (isCost) {
    return (
      <div className="text-xs text-neutral-600 px-10 py-0.5 font-body">
        {event.body}
      </div>
    );
  }

  const badgeClass = TOOL_BADGE[event.label] ?? "bg-neutral-800 text-neutral-300";

  return (
    <div className={`flex gap-3 px-4 py-2 ${isThink ? "opacity-60 italic" : ""}`}>
      <div
        className={`w-7 h-7 flex items-center justify-center text-[10px] font-bold shrink-0 ${badgeClass}`}
      >
        {initials(event.label)}
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className={`text-xs font-bold font-body ${badgeClass} px-1.5 py-0.5`}>
            {event.label}
          </span>
          <span className="text-xs text-neutral-600 font-body">{formatTime(event.at)}</span>
        </div>
        {event.body && (
          <div className="font-body whitespace-pre-wrap text-sm text-text-primary mt-1 break-words">
            {event.body}
          </div>
        )}
      </div>
    </div>
  );
}

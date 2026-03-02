import { useState } from "react";
import { ChevronRight } from "lucide-react";
import type { ToolGroupData } from "./group-tools";
import { MessageRow } from "./MessageRow";

const TOOL_BADGE_CLASS: Record<string, string> = {
  Read: "badge-read",
  Edit: "badge-edit",
  Write: "badge-write",
  Bash: "badge-bash",
  Glob: "badge-read",
  Grep: "badge-read",
};

function eventKey(event: { at: string; category: string; label: string; body: string }): string {
  return `${event.at}:${event.category}:${event.label}:${event.body}`;
}

export function ToolGroup({ group }: { group: ToolGroupData }) {
  const [open, setOpen] = useState(false);
  const badgeCls = TOOL_BADGE_CLASS[group.label] ?? "";

  return (
    <div className="tool-group">
      <button type="button" className="tool-group-summary" onClick={() => setOpen((v) => !v)} aria-expanded={open}>
        <span className={`msg-badge ${badgeCls}`}>{group.label}</span>
        <span className="tool-group-label">
          {group.label} (x{group.events.length})
        </span>
        <ChevronRight className={`tool-group-chevron ${open ? "open" : ""}`} size={14} />
      </button>
      <div className={`tool-group-children ${open ? "open" : ""}`}>
        <div>
          {group.events.map((event) => (
            <MessageRow key={eventKey(event)} event={event} hideBadge />
          ))}
        </div>
      </div>
    </div>
  );
}

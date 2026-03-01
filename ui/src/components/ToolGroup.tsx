import { useState } from "react";
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

function previewText(group: ToolGroupData): string {
  return group.events
    .map((e) => e.body.split("\n")[0].trim())
    .filter(Boolean)
    .join(", ");
}

function eventKey(event: { at: string; category: string; label: string; body: string }): string {
  return `${event.at}:${event.category}:${event.label}:${event.body}`;
}

export function ToolGroup({ group }: { group: ToolGroupData }) {
  const [open, setOpen] = useState(false);
  const badgeCls = TOOL_BADGE_CLASS[group.label] ?? "";

  return (
    <div className="tool-group">
      <button type="button" className="tool-group-summary" onClick={() => setOpen((v) => !v)}>
        <span className={`msg-badge ${badgeCls}`}>{group.label}</span>
        <span className="tool-group-label">
          {group.label} {group.events.length} files
        </span>
        <span className="tool-group-preview">{previewText(group)}</span>
        <span className={`tool-group-chevron ${open ? "open" : ""}`}>&#x25B8;</span>
      </button>
      {open && (
        <div className="tool-group-children">
          {group.events.map((event) => (
            <MessageRow key={eventKey(event)} event={event} />
          ))}
        </div>
      )}
    </div>
  );
}

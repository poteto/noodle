import type { EventLine } from "~/client";
import { MarkdownBody } from "./MarkdownBody";

const TOOL_BADGE_CLASS: Record<string, string> = {
  Read: "badge-read",
  Edit: "badge-edit",
  Write: "badge-write",
  Bash: "badge-bash",
  Glob: "badge-read",
  Grep: "badge-read",
};

const TOOL_LABELS = new Set(["Read", "Edit", "Write", "Bash", "Glob", "Grep"]);

const MARKDOWN_LABELS = new Set(["Think", "Prompt"]);

function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

function typeClass(event: EventLine): string {
  if (event.category === "ticket") {
    return "type-system";
  }
  if (event.label === "Cost") {
    return "type-cost";
  }
  if (event.label === "Think") {
    return "type-system";
  }
  if (TOOL_LABELS.has(event.label)) {
    return "type-tool";
  }
  if (event.label === "Scheduler") {
    return "from-manager";
  }
  if (event.label === "User") {
    return "from-user";
  }
  return "";
}

function badgeClass(event: EventLine): string {
  if (event.label === "User") {
    return "badge-user";
  }
  if (event.label === "Scheduler") {
    return "badge-manager";
  }
  if (TOOL_LABELS.has(event.label)) {
    return TOOL_BADGE_CLASS[event.label] ?? "";
  }
  // Agent names, Spawned, Think, etc. — default filled style
  return "badge-agent";
}

function BodyContent({ event }: { event: EventLine }) {
  if (!event.body) {
    return null;
  }

  if (MARKDOWN_LABELS.has(event.label)) {
    return <MarkdownBody text={event.body} />;
  }

  return <div className="msg-body">{event.body}</div>;
}

export function MessageRow({ event, hideBadge }: { event: EventLine; hideBadge?: boolean }) {
  if (event.category === "ticket") {
    return (
      <div className="idle-divider">
        <span>{event.label}</span>
      </div>
    );
  }

  // Spawned events render as section dividers
  if (event.label === "Spawned") {
    return (
      <div className="spawn-divider">
        <span>{event.body || "Session spawned"}</span>
      </div>
    );
  }

  if (event.label === "Cost") {
    return (
      <div className="message-row type-cost">
        <span className="msg-body" style={{ fontSize: 12 }}>
          {event.body}
        </span>
      </div>
    );
  }

  const badgeCls = badgeClass(event);
  const tc = typeClass(event);

  if (tc === "type-tool") {
    return (
      <div className={`message-row ${tc} tool-oneliner`}>
        {!hideBadge && <span className={`msg-badge ${badgeCls}`}>{event.label}</span>}
        <span className="tool-summary">{event.body.split("\n")[0].trim()}</span>
      </div>
    );
  }

  return (
    <div className={`message-row ${tc}`}>
      <div className="msg-meta">
        <span className={`msg-badge ${badgeCls}`}>{event.label}</span>
        <span>{formatTime(event.at)}</span>
      </div>
      <BodyContent event={event} />
    </div>
  );
}

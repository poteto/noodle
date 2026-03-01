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

const MARKDOWN_LABELS = new Set(["Think", "Prompt"]);

function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

function initials(label: string): string {
  return label.slice(0, 2).toUpperCase();
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
  if (event.label === "Manager") {
    return "from-manager";
  }
  if (event.label === "User") {
    return "from-user";
  }
  return "";
}

function tryFormatJson(text: string): string | null {
  try {
    const parsed = JSON.parse(text);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return null;
  }
}

function BodyContent({ event }: { event: EventLine }) {
  if (!event.body) {
    return null;
  }

  if (MARKDOWN_LABELS.has(event.label)) {
    return <MarkdownBody text={event.body} />;
  }

  if (event.label === "Spawned") {
    const formatted = tryFormatJson(event.body);
    if (formatted) {
      return <pre className="msg-body msg-json">{formatted}</pre>;
    }
  }

  return <div className="msg-body">{event.body}</div>;
}

export function MessageRow({ event }: { event: EventLine }) {
  if (event.category === "ticket") {
    return (
      <div className="idle-divider">
        <span>{event.label}</span>
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

  const badgeCls = TOOL_BADGE_CLASS[event.label] ?? "";

  return (
    <div className={`message-row ${typeClass(event)}`}>
      <div className="msg-avatar">{initials(event.label)}</div>
      <div>
        <div className="msg-meta">
          <span className={`msg-badge ${badgeCls}`}>{event.label}</span>
          <span>{formatTime(event.at)}</span>
        </div>
        <BodyContent event={event} />
      </div>
    </div>
  );
}

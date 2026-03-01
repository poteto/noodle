import type { EventLine } from "~/client";

const TOOL_LABELS = new Set(["Read", "Edit", "Write", "Bash", "Glob", "Grep"]);

export interface ToolGroupData {
  kind: "tool-group";
  label: string;
  events: EventLine[];
}

export function groupConsecutiveTools(events: EventLine[]): (EventLine | ToolGroupData)[] {
  const result: (EventLine | ToolGroupData)[] = [];
  let i = 0;

  while (i < events.length) {
    const ev = events[i];
    if (!TOOL_LABELS.has(ev.label)) {
      result.push(ev);
      i++;
      continue;
    }

    // Collect consecutive events with the same label
    const { label } = ev;
    const group: EventLine[] = [ev];
    let j = i + 1;
    while (j < events.length && events[j].label === label) {
      group.push(events[j]);
      j++;
    }

    if (group.length === 1) {
      result.push(ev);
    } else {
      result.push({ kind: "tool-group", label, events: group });
    }
    i = j;
  }

  return result;
}

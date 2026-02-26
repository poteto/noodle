import { useState } from "react";
import { useControl } from "./ControlContext";
import { Tooltip } from "./Tooltip";

export function ConcurrencyBadge({ active, maxCooks }: { active: number; maxCooks: number }) {
  const send = useControl();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(String(maxCooks));

  function startEdit() {
    setDraft(String(maxCooks));
    setEditing(true);
  }

  function commit() {
    const n = Number.parseInt(draft, 10);
    if (!Number.isNaN(n) && n >= 1 && n !== maxCooks) {
      send({ action: "set-max-cooks", value: String(n) });
    }
    setEditing(false);
  }

  if (editing) {
    return (
      <input
        className="font-mono text-xs font-bold w-12 px-1.5 py-0.5 border-[1.5px] border-nyellow bg-bg-1 text-text-0 outline-none text-center"
        type="number"
        min={1}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={commit}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            commit();
          }
          if (e.key === "Escape") {
            setEditing(false);
          }
        }}
        autoFocus
      />
    );
  }

  return (
    <Tooltip content="Click to edit max concurrency">
      <button
        type="button"
        className="font-mono text-xs font-bold px-2 py-0.5 bg-bg-1 border-[1.5px] border-border-subtle text-text-1 cursor-pointer transition-all duration-[0.12s] hover:border-border hover:bg-bg-hover"
        onClick={startEdit}
      >
        {active}/{maxCooks}
      </button>
    </Tooltip>
  );
}

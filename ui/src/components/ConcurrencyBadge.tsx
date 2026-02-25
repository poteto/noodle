import { useState } from "react";
import { useSendControl } from "~/client";

export function ConcurrencyBadge({
  active,
  maxCooks,
}: {
  active: number;
  maxCooks: number;
}) {
  const { mutate: send } = useSendControl();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(String(maxCooks));

  function startEdit() {
    setDraft(String(maxCooks));
    setEditing(true);
  }

  function commit() {
    const n = parseInt(draft, 10);
    if (!isNaN(n) && n >= 1 && n !== maxCooks) {
      send({ action: "set-max-cooks", value: String(n) });
    }
    setEditing(false);
  }

  if (editing) {
    return (
      <input
        className="concurrency-input"
        type="number"
        min={1}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={commit}
        onKeyDown={(e) => {
          if (e.key === "Enter") commit();
          if (e.key === "Escape") setEditing(false);
        }}
        autoFocus
      />
    );
  }

  return (
    <button
      className="concurrency-badge"
      onClick={startEdit}
      title="Click to edit max concurrency"
    >
      {active}/{maxCooks}
    </button>
  );
}

import { useState } from "react";

export function ConfigWarnings({ warnings }: { warnings: string[] }) {
  const [dismissed, setDismissed] = useState(false);

  if (dismissed || warnings.length === 0) {
    return null;
  }

  return (
    <div className="mx-10 mt-4 bg-nyellow-bg border-l-4 border-l-nyellow px-4 py-3 flex items-start gap-3">
      <div className="flex-1 font-body text-[0.8125rem] text-text-0">
        {warnings.map((msg, i) => (
          <p key={i}>{msg}</p>
        ))}
      </div>
      <button
        type="button"
        onClick={() => setDismissed(true)}
        className="text-text-3 hover:text-text-0 font-mono text-xs cursor-pointer shrink-0"
      >
        dismiss
      </button>
    </div>
  );
}

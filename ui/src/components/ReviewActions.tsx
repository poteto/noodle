import { useState } from "react";
import { useControl } from "./ControlContext";

export function ReviewActions({ itemId }: { itemId: string }) {
  const send = useControl();
  const [showFeedback, setShowFeedback] = useState(false);
  const [feedback, setFeedback] = useState("");

  function handleMerge() {
    send({ action: "merge", item: itemId });
  }

  function handleReject() {
    send({ action: "reject", item: itemId });
  }

  function handleRequestChanges() {
    if (!showFeedback) {
      setShowFeedback(true);
      return;
    }
    send({
      action: "request-changes",
      item: itemId,
      prompt: feedback,
    });
  }

  return (
    <>
      {showFeedback && (
        <div className="mt-2 mb-1">
          <input
            type="text"
            className="w-full px-[10px] py-[6px] font-body text-[0.8125rem] border-2 border-border bg-bg-1 text-text-0 outline-none focus:border-nyellow"
            placeholder="What needs to change?"
            value={feedback}
            onChange={(e) => setFeedback(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") handleRequestChanges();
              if (e.key === "Escape") setShowFeedback(false);
            }}
            autoFocus
          />
        </div>
      )}
      <div className="flex gap-[6px] mt-[10px]">
        <button
          className="flex-1 flex items-center justify-center gap-1 py-[5px] px-[10px] font-body text-xs font-bold border-2 cursor-pointer transition-all duration-[0.12s] bg-ngreen text-white border-ngreen hover:brightness-110"
          onClick={handleMerge}
        >
          merge
        </button>
        <button
          className="flex-1 flex items-center justify-center gap-1 py-[5px] px-[10px] font-body text-xs font-bold border-2 cursor-pointer transition-all duration-[0.12s] bg-nyellow-bg border-nyellow-dim text-nyellow hover:brightness-95"
          onClick={handleRequestChanges}
        >
          {showFeedback ? "send" : "changes"}
        </button>
        <button
          className="flex-1 flex items-center justify-center gap-1 py-[5px] px-[10px] font-body text-xs font-bold border-2 border-border bg-bg-1 text-text-1 cursor-pointer transition-all duration-[0.12s] hover:bg-nred-dim hover:border-nred hover:text-nred"
          onClick={handleReject}
        >
          reject
        </button>
      </div>
    </>
  );
}

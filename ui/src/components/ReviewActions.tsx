import { useState } from "react";
import type { ControlCommand } from "~/client";
import { useSendControl } from "~/client";

export function ReviewActions({
  itemId,
  onOptimistic,
  onRevert,
}: {
  itemId: string;
  onOptimistic: () => void;
  onRevert: () => void;
}) {
  const { mutate: send, isPending } = useSendControl();
  const [showFeedback, setShowFeedback] = useState(false);
  const [feedback, setFeedback] = useState("");

  function dispatch(cmd: ControlCommand) {
    onOptimistic();
    send(cmd, {
      onError: () => onRevert(),
    });
  }

  function handleMerge() {
    dispatch({ action: "merge", item: itemId });
  }

  function handleReject() {
    dispatch({ action: "reject", item: itemId });
  }

  function handleRequestChanges() {
    if (!showFeedback) {
      setShowFeedback(true);
      return;
    }
    dispatch({
      action: "request-changes",
      item: itemId,
      prompt: feedback,
    });
  }

  return (
    <>
      {showFeedback && (
        <div className="review-feedback">
          <input
            type="text"
            className="review-feedback-input"
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
      <div className="card-review-actions">
        <button
          className="merge-btn"
          onClick={handleMerge}
          disabled={isPending}
        >
          merge
        </button>
        <button
          className="changes-btn"
          onClick={handleRequestChanges}
          disabled={isPending}
        >
          {showFeedback ? "send" : "changes"}
        </button>
        <button
          className="reject-btn"
          onClick={handleReject}
          disabled={isPending}
        >
          reject
        </button>
      </div>
    </>
  );
}

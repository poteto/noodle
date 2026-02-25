import { useState } from "react";
import { useSendControl } from "~/client";

export function ReviewActions({ itemId }: { itemId: string }) {
  const { mutate: send, isPending } = useSendControl();
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
        <button className="merge-btn" onClick={handleMerge} disabled={isPending}>
          merge
        </button>
        <button className="changes-btn" onClick={handleRequestChanges} disabled={isPending}>
          {showFeedback ? "send" : "changes"}
        </button>
        <button className="reject-btn" onClick={handleReject} disabled={isPending}>
          reject
        </button>
      </div>
    </>
  );
}

import { useState } from "react";
import { useSendControl } from "~/client";
import type { PendingReviewItem } from "~/client";

export function ReviewBanner({ review }: { review: PendingReviewItem }) {
  const { mutate: send, isPending } = useSendControl();
  const [showFeedback, setShowFeedback] = useState(false);
  const [feedback, setFeedback] = useState("");

  function handleMerge() {
    send({ action: "merge", order_id: review.order_id });
  }

  function handleReject() {
    send({ action: "reject", order_id: review.order_id });
  }

  function handleRequestChanges() {
    if (!showFeedback) {
      setShowFeedback(true);
      return;
    }
    send({
      action: "request-changes",
      order_id: review.order_id,
      prompt: feedback,
    });
    setShowFeedback(false);
    setFeedback("");
  }

  return (
    <div className="border-l-2 border-accent p-4 bg-accent/5 mx-4 mb-2">
      <div className="text-xs font-display font-bold uppercase tracking-wider text-accent mb-1">
        Ready for review
      </div>
      {review.reason && (
        <div className="text-xs font-body text-neutral-400 mb-3">{review.reason}</div>
      )}

      {showFeedback && (
        <div className="mb-3">
          <input
            type="text"
            className="w-full px-2 py-1.5 font-body text-xs border border-border-subtle bg-transparent text-text-primary outline-none focus:border-accent"
            placeholder="What needs to change?"
            value={feedback}
            onChange={(e) => setFeedback(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                handleRequestChanges();
              }
              if (e.key === "Escape") {
                setShowFeedback(false);
              }
            }}
            autoFocus
          />
        </div>
      )}

      <div className="flex gap-2">
        <button
          type="button"
          onClick={handleMerge}
          disabled={isPending}
          className="font-body font-bold uppercase text-[10px] tracking-wider px-3 py-1.5 bg-green text-black"
        >
          {isPending ? "MERGING…" : "MERGE"}
        </button>
        <button
          type="button"
          onClick={handleReject}
          disabled={isPending}
          className="font-body font-bold uppercase text-[10px] tracking-wider px-3 py-1.5 bg-red text-white"
        >
          REJECT
        </button>
        <button
          type="button"
          onClick={handleRequestChanges}
          disabled={isPending}
          className="font-body font-bold uppercase text-[10px] tracking-wider px-3 py-1.5 border border-accent text-accent"
        >
          {showFeedback ? "SEND" : "REQUEST CHANGES"}
        </button>
      </div>
    </div>
  );
}

import { useState } from "react";
import { useSuspenseSnapshot, useSendControl, useReviewDiff } from "~/client";
import type { PendingReviewItem } from "~/client";
import { DiffViewer } from "./DiffViewer";

function ReviewReport({ review }: { review: PendingReviewItem }) {
  const { data, isLoading, error } = useReviewDiff(review.order_id);

  return (
    <main className="review-report">
      <header className="feed-header">
        <div className="feed-title">
          Quality Review:{" "}
          <span className="review-session-name">{review.session_id || review.order_id}</span>
          {review.task_key && <span className="feed-badge badge-task">{review.task_key}</span>}
        </div>
        {review.reason && (
          <span
            style={{
              fontFamily: "var(--font-mono)",
              fontSize: 12,
              color: "var(--color-text-tertiary)",
            }}
          >
            {review.reason}
          </span>
        )}
      </header>

      <section className="review-section">
        <h2 className="review-section-title">Generated Diff</h2>
        <DiffViewer
          diff={data?.diff ?? ""}
          stat={data?.stat ?? ""}
          isLoading={isLoading}
          error={error?.message}
        />
      </section>
    </main>
  );
}

function ReviewActions({ review }: { review: PendingReviewItem }) {
  const { mutate: send, isPending } = useSendControl();
  const [showFeedback, setShowFeedback] = useState(false);
  const [feedback, setFeedback] = useState("");

  function handleMerge() {
    send({ action: "merge", order_id: review.order_id });
  }

  function handleRequestChanges() {
    if (!showFeedback) {
      setShowFeedback(true);
      return;
    }
    if (feedback.trim()) {
      send({ action: "request-changes", order_id: review.order_id, prompt: feedback });
      setShowFeedback(false);
      setFeedback("");
    }
  }

  function handleReject() {
    send({ action: "reject", order_id: review.order_id });
  }

  let requestChangesLabel = "Request Changes";
  if (isPending) {
    requestChangesLabel = "Sending…";
  } else if (showFeedback) {
    requestChangesLabel = "Send";
  }

  return (
    <aside className="context-panel">
      <div className="context-header">Review Actions</div>

      <div className="review-actions-body">
        {review.skill && (
          <div className="review-meta-item">
            <span className="review-meta-label">Skill</span>
            <span className="review-meta-value">{review.skill}</span>
          </div>
        )}
        {review.model && (
          <div className="review-meta-item">
            <span className="review-meta-label">Model</span>
            <span className="review-meta-value">{review.model}</span>
          </div>
        )}
        {review.worktree_name && (
          <div className="review-meta-item">
            <span className="review-meta-label">Worktree</span>
            <span className="review-meta-value">{review.worktree_name}</span>
          </div>
        )}
      </div>

      <div style={{ flex: 1 }} />

      <div className="review-action-footer">
        {showFeedback && (
          <input
            type="text"
            className="review-feedback-input"
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
        )}
        <button
          type="button"
          className="review-action-btn review-action-primary"
          onClick={handleMerge}
          disabled={isPending}
        >
          {isPending ? "Merging…" : "Approve & Merge"}
        </button>
        <div className="review-action-row">
          <button
            type="button"
            className="review-action-btn review-action-outline"
            onClick={handleRequestChanges}
            disabled={isPending}
          >
            {requestChangesLabel}
          </button>
          <button
            type="button"
            className="review-action-btn review-action-danger"
            onClick={handleReject}
            disabled={isPending}
          >
            {isPending ? "Dismissing…" : "Dismiss Order"}
          </button>
        </div>
      </div>
    </aside>
  );
}

function EmptyReviews() {
  return (
    <>
      <main className="review-report">
        <header className="feed-header">
          <div className="feed-title">Reviews</div>
        </header>
        <div className="review-empty">No pending reviews. Completed sessions in supervised mode appear here for review.</div>
      </main>
      <aside className="context-panel">
        <div className="context-header">Review Actions</div>
      </aside>
    </>
  );
}

export function ReviewList() {
  const { data: snapshot } = useSuspenseSnapshot();
  const reviews = snapshot.pending_reviews ?? [];
  const [selectedIdx, setSelectedIdx] = useState(0);

  if (reviews.length === 0) {
    return (
      <div className="grid grid-cols-[1fr_300px] h-full overflow-hidden">
        <EmptyReviews />
      </div>
    );
  }

  const activeIdx = selectedIdx >= 0 && selectedIdx < reviews.length ? selectedIdx : 0;
  const review = reviews[activeIdx];
  if (!review) {
    return (
      <div className="grid grid-cols-[1fr_300px] h-full overflow-hidden">
        <EmptyReviews />
      </div>
    );
  }

  return (
    <div className="grid grid-cols-[1fr_300px] h-full overflow-hidden">
      {reviews.length > 1 && (
        <div className="review-tab-bar" role="tablist">
          {reviews.map((r, i) => (
            <button
              key={r.order_id}
              type="button"
              role="tab"
              aria-selected={i === activeIdx}
              className={`review-tab ${i === activeIdx ? "active" : ""}`}
              onClick={() => setSelectedIdx(i)}
            >
              {r.title || r.task_key || r.order_id}
            </button>
          ))}
        </div>
      )}
      <ReviewReport review={review} />
      <ReviewActions review={review} />
    </div>
  );
}

import { useState } from "react";
import { useSuspenseSnapshot, useSendControl, useReviewDiff } from "~/client";
import type { PendingReviewItem } from "~/client";
import { DiffViewer } from "./DiffViewer";

function ReviewReport({ review }: { review: PendingReviewItem }) {
  const { data, isLoading, error } = useReviewDiff(review.order_id);

  return (
    <main className="review-report">
      <div className="review-report-header">
        <div className="review-report-title-row">
          <h1 className="review-report-title">
            Quality Review: <span className="review-session-name">{review.session_id || review.order_id}</span>
          </h1>
          {review.task_key && <span className="feed-badge badge-task">{review.task_key}</span>}
        </div>
        {review.reason && <p className="review-report-subtitle">{review.reason}</p>}
      </div>

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
  const { mutate: send } = useSendControl();
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
              if (e.key === "Enter") handleRequestChanges();
              if (e.key === "Escape") setShowFeedback(false);
            }}
            autoFocus
          />
        )}
        <button type="button" className="review-action-btn review-action-primary" onClick={handleMerge}>
          Approve &amp; Merge
        </button>
        <div className="review-action-row">
          <button type="button" className="review-action-btn review-action-outline" onClick={handleRequestChanges}>
            {showFeedback ? "Send" : "Request Changes"}
          </button>
          <button type="button" className="review-action-btn review-action-danger" onClick={handleReject}>
            Dismiss Order
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
        <div className="review-report-header">
          <h1 className="review-report-title">Reviews</h1>
        </div>
        <div className="review-empty">No pending reviews</div>
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
  const review = reviews[selectedIdx];

  if (reviews.length === 0) {
    return (
      <div className="grid grid-cols-[1fr_300px] h-full overflow-hidden">
        <EmptyReviews />
      </div>
    );
  }

  return (
    <div className="grid grid-cols-[1fr_300px] h-full overflow-hidden">
      {reviews.length > 1 && (
        <div className="review-tab-bar">
          {reviews.map((r, i) => (
            <button
              key={r.order_id}
              type="button"
              className={`review-tab ${i === selectedIdx ? "active" : ""}`}
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

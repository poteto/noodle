export function SkeletonCard() {
  return (
    <div className="board-card schedule-card skeleton-card">
      <div className="card-top">
        <span className="skeleton-pill skeleton-badge" />
      </div>
      <div className="skeleton-pill skeleton-title" />
      <div className="skeleton-pill skeleton-text" />
      <div className="skeleton-pill skeleton-text short" />
      <div className="card-footer">
        <span className="skeleton-pill skeleton-model" />
      </div>
    </div>
  );
}

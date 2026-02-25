import { SkeletonCard } from "./SkeletonCard";

export function BoardSkeleton() {
  return (
    <div className="board-shell">
      <div className="board-header">
        <div className="board-header-left">
          <h1 className="board-title">noodle</h1>
        </div>
      </div>
      <div className="board-columns">
        <div className="board-col">
          <div className="col-header">
            <div className="col-header-inner">
              <span className="col-title">Queued</span>
              <div className="col-header-right">
                <span className="col-count">0</span>
              </div>
            </div>
          </div>
          <div className="col-cards">
            <SkeletonCard />
          </div>
        </div>
        <div className="board-col">
          <div className="col-header">
            <div className="col-header-inner">
              <span className="col-title">Cooking</span>
              <div className="col-header-right">
                <span className="col-count">0</span>
              </div>
            </div>
          </div>
          <div className="col-cards">
            <div className="col-empty">No active cooks</div>
          </div>
        </div>
        <div className="board-col">
          <div className="col-header">
            <div className="col-header-inner">
              <span className="col-title">Review</span>
              <div className="col-header-right">
                <span className="col-count">0</span>
              </div>
            </div>
          </div>
          <div className="col-cards">
            <div className="col-empty">Nothing to review</div>
          </div>
        </div>
        <div className="board-col">
          <div className="col-header">
            <div className="col-header-inner">
              <span className="col-title">Done</span>
              <div className="col-header-right">
                <span className="col-count">0</span>
              </div>
            </div>
          </div>
          <div className="col-cards">
            <div className="col-empty">No completed tasks</div>
          </div>
        </div>
      </div>
    </div>
  );
}

import { useRef, useState } from "react";
import type { KeyboardEvent } from "react";
import { useSuspenseSnapshot, useSendControl } from "~/client";
import type { BacklogItem } from "~/client";

// Enter confirms an IME composition before it submits anything.
function isSubmitKey(e: KeyboardEvent<HTMLInputElement>) {
  return e.key === "Enter" && !e.nativeEvent.isComposing;
}

function AddItem() {
  const { mutate: send, isPending } = useSendControl();
  const [title, setTitle] = useState("");

  function submit() {
    const trimmed = title.trim();
    if (!trimmed) {
      return;
    }
    send({ action: "backlog-add", prompt: trimmed });
    setTitle("");
  }

  return (
    <div className="backlog-add">
      <input
        type="text"
        className="review-feedback-input"
        placeholder="Add a task to the backlog…"
        aria-label="New backlog item"
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        onKeyDown={(e) => {
          if (isSubmitKey(e)) {
            submit();
          }
        }}
      />
      <button
        type="button"
        className="review-action-btn review-action-primary"
        onClick={submit}
        disabled={isPending || !title.trim()}
      >
        Add
      </button>
    </div>
  );
}

function ItemMeta({ item, showSection }: { item: BacklogItem; showSection: boolean }) {
  return (
    <>
      {item.status === "in_progress" && (
        <span className="feed-badge badge-status">in progress</span>
      )}
      {showSection && item.section && <span className="feed-badge">{item.section}</span>}
      {item.tags.map((tag) => (
        <span key={tag} className="feed-badge badge-task">
          #{tag}
        </span>
      ))}
      {item.estimate && <span className="feed-badge">~{item.estimate}</span>}
    </>
  );
}

function OpenItem({ item, showSection }: { item: BacklogItem; showSection: boolean }) {
  const { mutate: send, isPending } = useSendControl();
  const [draft, setDraft] = useState<string | null>(null);
  const cancelled = useRef(false);

  // Blur is the only commit path. Enter and Escape blur the field rather than
  // committing themselves, so the blur a browser may fire when the field
  // unmounts can't save twice or save a discarded edit.
  function finishEdit() {
    const trimmed = draft?.trim() ?? "";
    const discard = cancelled.current;
    cancelled.current = false;
    setDraft(null);
    if (!discard && trimmed && trimmed !== item.title) {
      send({ action: "backlog-edit", target: item.id, prompt: trimmed });
    }
  }

  return (
    <li className="backlog-item">
      <span className="backlog-id">#{item.id}</span>
      {draft === null ? (
        <button
          type="button"
          className="backlog-title"
          onClick={() => setDraft(item.title)}
          aria-label={`Edit #${item.id}`}
          title="Click to edit"
        >
          {item.title}
        </button>
      ) : (
        <input
          type="text"
          className="review-feedback-input backlog-edit-input"
          aria-label={`Title for #${item.id}`}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={finishEdit}
          onKeyDown={(e) => {
            if (isSubmitKey(e)) {
              e.currentTarget.blur();
            }
            if (e.key === "Escape") {
              cancelled.current = true;
              e.currentTarget.blur();
            }
          }}
          autoFocus
        />
      )}
      <span className="backlog-meta">
        <ItemMeta item={item} showSection={showSection} />
      </span>
      <button
        type="button"
        className="review-action-btn review-action-outline backlog-done-btn"
        onClick={() => send({ action: "backlog-done", target: item.id })}
        disabled={isPending}
        aria-label={`Mark #${item.id} done`}
      >
        Done
      </button>
    </li>
  );
}

// The backlog the scheduler sees: mise drops done items, so marking one done
// takes it off this list.
export function BacklogView() {
  const { data: snapshot } = useSuspenseSnapshot();
  const backlog = snapshot.backlog ?? [];
  // A section label on every row is noise when everything shares one.
  const showSection = new Set(backlog.map((item) => item.section)).size > 1;

  return (
    <main className="backlog-page">
      <header className="feed-header">
        <div className="feed-title">
          Backlog <span className="feed-badge">{backlog.length} open</span>
        </div>
      </header>

      {snapshot.backlog_error && (
        <div className="backlog-error" role="alert">
          Last backlog change failed: {snapshot.backlog_error}
        </div>
      )}

      <AddItem />

      {backlog.length === 0 ? (
        <div className="review-empty">
          The backlog is empty. Add a task above; the scheduler picks it up on its next cycle.
        </div>
      ) : (
        <ul className="backlog-list" aria-label="Open items">
          {backlog.map((item) => (
            <OpenItem key={item.id} item={item} showSection={showSection} />
          ))}
        </ul>
      )}
    </main>
  );
}

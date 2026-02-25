import { useState, useRef, useEffect } from "react";
import { useSendControl, useConfig } from "~/client";

export function QueueAddCard() {
  const [open, setOpen] = useState(false);
  const [prompt, setPrompt] = useState("");
  const { data: config } = useConfig();
  const { mutate: send, isPending } = useSendControl();
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (open && textareaRef.current) {
      textareaRef.current.focus();
    }
  }, [open]);

  function handleSubmit() {
    const text = prompt.trim();
    if (!text) return;
    send(
      {
        action: "enqueue",
        item: `task-${Date.now()}`,
        prompt: text,
        task_key: "execute",
        provider: config?.provider,
        model: config?.model,
      },
      {
        onSuccess: () => {
          setPrompt("");
          setOpen(false);
        },
      },
    );
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      handleSubmit();
    }
    if (e.key === "Escape") {
      setPrompt("");
      setOpen(false);
    }
  }

  if (!open) {
    return (
      <button className="queue-add-btn" onClick={() => setOpen(true)}>
        + add task
      </button>
    );
  }

  return (
    <div className="queue-add-form">
      <textarea
        ref={textareaRef}
        className="queue-add-textarea"
        value={prompt}
        onChange={(e) => setPrompt(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="What should the agent do?"
        rows={3}
      />
      <div className="queue-add-actions">
        <button
          className="queue-add-submit"
          onClick={handleSubmit}
          disabled={isPending || !prompt.trim()}
        >
          add
        </button>
        <button
          className="queue-add-cancel"
          onClick={() => {
            setPrompt("");
            setOpen(false);
          }}
        >
          x
        </button>
      </div>
    </div>
  );
}

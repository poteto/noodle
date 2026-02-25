import { useState } from "react";
import { useSendControl, useConfig } from "~/client";

export function QueueAddCard() {
  const [open, setOpen] = useState(false);
  const [prompt, setPrompt] = useState("");
  const { data: config } = useConfig();
  const { mutate: send, isPending } = useSendControl();

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
      <button
        className="w-full py-[10px] px-[14px] font-body text-[0.8125rem] font-semibold text-text-2 bg-transparent border-2 border-dashed border-border-subtle cursor-pointer transition-all duration-[0.12s] hover:text-text-0 hover:border-border hover:bg-bg-1"
        onClick={() => setOpen(true)}
      >
        + add task
      </button>
    );
  }

  return (
    <div className="bg-bg-1 border-2 border-border p-3 shadow-card">
      <textarea
        className="w-full py-2 px-[10px] font-body text-[0.8125rem] border-2 border-border bg-bg-1 text-text-0 resize-y min-h-[60px] outline-none focus:border-nyellow"
        autoFocus
        value={prompt}
        onChange={(e) => setPrompt(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="What should the agent do?"
        rows={3}
      />
      <div className="flex items-center gap-2 mt-2">
        <button
          className="py-[5px] px-4 font-display text-[0.8125rem] font-bold bg-border text-bg-0 border-2 border-border cursor-pointer hover:enabled:brightness-120 disabled:opacity-40 disabled:cursor-not-allowed"
          onClick={handleSubmit}
          disabled={isPending || !prompt.trim()}
        >
          add
        </button>
        <button
          className="py-[3px] px-[10px] font-mono text-[0.8125rem] font-bold bg-transparent border-none text-text-2 cursor-pointer hover:text-text-0"
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

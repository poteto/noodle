import { useState, useEffect, useRef } from "react";
import { useSendControl, useConfig } from "~/client";
import type { ControlCommand } from "~/client";

export function TaskEditor({ onClose, editItemId }: { onClose: () => void; editItemId?: string }) {
  const { data: config } = useConfig();
  const { mutate: send, isPending } = useSendControl();
  const backdropRef = useRef<HTMLDivElement>(null);

  const [prompt, setPrompt] = useState("");
  const [taskKey, setTaskKey] = useState("execute");
  const [providerOverride, setProviderOverride] = useState<string | null>(null);
  const [modelOverride, setModelOverride] = useState<string | null>(null);
  // Derive from config, allow user override
  const provider = providerOverride ?? config?.provider ?? "";
  const model = modelOverride ?? config?.model ?? "";

  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") {
        onClose();
      }
    }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [onClose]);

  function handleBackdropClick(e: React.MouseEvent) {
    if (e.target === backdropRef.current) {
      onClose();
    }
  }

  function handleSubmit() {
    const text = prompt.trim();
    if (!text) {
      return;
    }

    const cmd: ControlCommand = editItemId
      ? {
          action: "edit-item",
          item: editItemId,
          prompt: text,
          task_key: taskKey,
          provider,
          model,
        }
      : {
          action: "enqueue",
          item: `task-${Date.now()}`,
          prompt: text,
          task_key: taskKey,
          provider,
          model,
        };

    send(cmd, { onSuccess: () => onClose() });
  }

  return (
    <div
      className="fixed inset-0 bg-[rgba(26,20,0,0.3)] z-100 flex items-center justify-center animate-fade-in"
      ref={backdropRef}
      role="presentation"
      onClick={handleBackdropClick}
      onKeyDown={(e) => {
        if (e.key === "Escape") {
          onClose();
        }
      }}
    >
      <div className="w-[520px] max-w-[90vw] bg-bg-1 border-3 border-border shadow-modal animate-scale-in">
        <div className="flex items-center justify-between px-5 py-4 border-b-2 border-border">
          <span className="font-display font-extrabold text-2xl">
            {editItemId ? "Edit Task" : "New Task"}
          </span>
          <button
            type="button"
            className="bg-transparent border-2 border-border px-2.5 py-0.5 font-mono text-[0.8125rem] font-bold cursor-pointer text-text-1 hover:bg-bg-hover active:translate-x-px active:translate-y-px active:shadow-btn-active"
            onClick={onClose}
          >
            x
          </button>
        </div>

        <div className="p-5 flex flex-col gap-3">
          <label
            htmlFor="task-prompt"
            className="font-mono text-xs font-semibold text-text-2 block mb-1"
          >
            Prompt
          </label>
          <textarea
            id="task-prompt"
            className="w-full px-3 py-2 font-body text-[0.8125rem] border-2 border-border bg-bg-1 text-text-0 resize-y min-h-[80px] outline-none focus:border-nyellow"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="What should the agent do?"
            rows={4}
            autoFocus
          />

          <div className="flex gap-3">
            <div className="flex-1">
              <label
                htmlFor="task-type"
                className="font-mono text-xs font-semibold text-text-2 block mb-1"
              >
                Type
              </label>
              <select
                id="task-type"
                className="w-full px-2.5 py-1.5 font-mono text-xs border-2 border-border bg-bg-1 text-text-0 outline-none focus:border-nyellow"
                value={taskKey}
                onChange={(e) => setTaskKey(e.target.value)}
              >
                <option value="execute">execute</option>
                <option value="review">review</option>
                <option value="plan">plan</option>
                <option value="reflect">reflect</option>
                <option value="schedule">schedule</option>
              </select>
            </div>

            <div className="flex-1">
              <label
                htmlFor="task-provider"
                className="font-mono text-xs font-semibold text-text-2 block mb-1"
              >
                Provider
              </label>
              <select
                id="task-provider"
                className="w-full px-2.5 py-1.5 font-mono text-xs border-2 border-border bg-bg-1 text-text-0 outline-none focus:border-nyellow"
                value={provider}
                onChange={(e) => setProviderOverride(e.target.value)}
              >
                <option value="claude">claude</option>
                <option value="codex">codex</option>
              </select>
            </div>

            <div className="flex-1">
              <label
                htmlFor="task-model"
                className="font-mono text-xs font-semibold text-text-2 block mb-1"
              >
                Model
              </label>
              <input
                id="task-model"
                className="w-full px-2.5 py-1.5 font-mono text-xs border-2 border-border bg-bg-1 text-text-0 outline-none focus:border-nyellow"
                value={model}
                onChange={(e) => setModelOverride(e.target.value)}
              />
            </div>
          </div>
        </div>

        <div className="flex justify-end gap-2 px-5 py-4 border-t-2 border-border">
          <button
            type="button"
            className="px-4 py-1.5 font-body text-[0.8125rem] font-semibold bg-bg-1 text-text-1 border-2 border-border cursor-pointer hover:bg-bg-hover active:translate-x-px active:translate-y-px active:shadow-btn-active"
            onClick={onClose}
          >
            cancel
          </button>
          <button
            type="button"
            className="px-5 py-1.5 font-display text-[0.8125rem] font-bold bg-accent text-bg-0 border-2 border-border cursor-pointer hover:not-disabled:brightness-120 disabled:opacity-40 disabled:cursor-not-allowed active:translate-x-px active:translate-y-px active:shadow-btn-active"
            onClick={handleSubmit}
            disabled={isPending || !prompt.trim()}
          >
            {editItemId ? "save" : "enqueue"}
          </button>
        </div>
      </div>
    </div>
  );
}

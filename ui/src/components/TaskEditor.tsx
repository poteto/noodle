import { useState, useEffect, useRef } from "react";
import { useSendControl, useConfig } from "~/client";
import type { ControlCommand } from "~/client";

export function TaskEditor({
  onClose,
  editItemId,
}: {
  onClose: () => void;
  editItemId?: string;
}) {
  const { data: config } = useConfig();
  const { mutate: send, isPending } = useSendControl();
  const backdropRef = useRef<HTMLDivElement>(null);

  const [prompt, setPrompt] = useState("");
  const [taskKey, setTaskKey] = useState("execute");
  const [provider, setProvider] = useState("");
  const [model, setModel] = useState("");
  const [skill, setSkill] = useState("");

  useEffect(() => {
    if (config) {
      setProvider(config.provider);
      setModel(config.model);
    }
  }, [config]);

  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [onClose]);

  function handleBackdropClick(e: React.MouseEvent) {
    if (e.target === backdropRef.current) onClose();
  }

  function handleSubmit() {
    const text = prompt.trim();
    if (!text) return;

    const cmd: ControlCommand = editItemId
      ? {
          action: "edit-item",
          item: editItemId,
          prompt: text,
          task_key: taskKey,
          provider,
          model,
          skill,
        }
      : {
          action: "enqueue",
          item: `task-${Date.now()}`,
          prompt: text,
          task_key: taskKey,
          provider,
          model,
          skill,
        };

    send(cmd, { onSuccess: () => onClose() });
  }

  return (
    <div className="task-editor-backdrop" ref={backdropRef} onClick={handleBackdropClick}>
      <div className="task-editor">
        <div className="task-editor-header">
          <span className="task-editor-title">
            {editItemId ? "Edit Task" : "New Task"}
          </span>
          <button className="chat-close" onClick={onClose}>
            x
          </button>
        </div>

        <div className="task-editor-body">
          <label className="task-editor-label">Prompt</label>
          <textarea
            className="task-editor-textarea"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="What should the agent do?"
            rows={4}
            autoFocus
          />

          <div className="task-editor-row">
            <div className="task-editor-field">
              <label className="task-editor-label">Type</label>
              <select
                className="task-editor-select"
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

            <div className="task-editor-field">
              <label className="task-editor-label">Provider</label>
              <select
                className="task-editor-select"
                value={provider}
                onChange={(e) => setProvider(e.target.value)}
              >
                <option value="claude">claude</option>
                <option value="codex">codex</option>
              </select>
            </div>

            <div className="task-editor-field">
              <label className="task-editor-label">Model</label>
              <input
                className="task-editor-input"
                value={model}
                onChange={(e) => setModel(e.target.value)}
              />
            </div>
          </div>

          <div className="task-editor-field">
            <label className="task-editor-label">Skill (optional)</label>
            <input
              className="task-editor-input"
              value={skill}
              onChange={(e) => setSkill(e.target.value)}
              placeholder="e.g., execute"
            />
          </div>
        </div>

        <div className="task-editor-footer">
          <button className="task-editor-cancel" onClick={onClose}>
            cancel
          </button>
          <button
            className="task-editor-submit"
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

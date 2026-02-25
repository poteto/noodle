import { useRef, useState } from "react";
import { useSendControl } from "~/client";

export function ChatInput({ sessionId }: { sessionId: string }) {
  const [value, setValue] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const { mutate: send, isPending } = useSendControl();

  function handleSubmit() {
    const text = value.trim();
    if (!text) return;
    send(
      { action: "steer", target: sessionId, prompt: text },
      { onSuccess: () => setValue("") },
    );
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  }

  function handleInput() {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 120) + "px";
  }

  return (
    <div className="chat-input-bar">
      <textarea
        ref={textareaRef}
        className="chat-textarea"
        placeholder="Steer this agent..."
        value={value}
        onChange={(e) => {
          setValue(e.target.value);
          handleInput();
        }}
        onKeyDown={handleKeyDown}
        rows={1}
        disabled={isPending}
      />
      <button
        className="chat-send-btn"
        onClick={handleSubmit}
        disabled={isPending || !value.trim()}
      >
        send
      </button>
    </div>
  );
}

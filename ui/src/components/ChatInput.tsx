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
    <div className="flex gap-2 px-5 py-3 border-t-2 border-border bg-bg-1 shrink-0">
      <textarea
        ref={textareaRef}
        className="flex-1 resize-none px-3 py-2 font-body text-[0.8125rem] border-2 border-border bg-bg-1 text-text-0 outline-none min-h-9 max-h-[120px] focus:border-nyellow"
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
        className="px-4 py-1.5 font-display text-[0.8125rem] font-bold bg-accent text-bg-0 border-2 border-border cursor-pointer self-end [&:hover:not(:disabled)]:brightness-[1.2] disabled:opacity-40 disabled:cursor-not-allowed [&:active:not(:disabled)]:translate-x-px [&:active:not(:disabled)]:translate-y-px [&:active:not(:disabled)]:shadow-btn-active"
        onClick={handleSubmit}
        disabled={isPending || !value.trim()}
      >
        send
      </button>
    </div>
  );
}

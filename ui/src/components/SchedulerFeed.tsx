import { useState } from "react";
import { useSuspenseSnapshot, useSendControl, formatCost } from "~/client";

export function SchedulerFeed() {
  const { data: snapshot } = useSuspenseSnapshot();
  const { mutate: send, isPending } = useSendControl();
  const [input, setInput] = useState("");

  function handleSubmit() {
    const prompt = input.trim();
    if (!prompt) return;
    send({ action: "steer", name: "schedule", prompt });
    setInput("");
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="p-4 border-b border-border-subtle">
        <h2 className="text-sm font-display font-bold uppercase tracking-wider">SCHEDULER</h2>
        <p className="text-xs text-neutral-500 mt-1 font-body">
          {snapshot.loop_state} &middot; {snapshot.orders.length} order{snapshot.orders.length !== 1 ? "s" : ""} &middot; {formatCost(snapshot.total_cost_usd)}
        </p>
      </div>

      {/* Order summary list */}
      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        {snapshot.orders.length === 0 && (
          <p className="text-sm text-neutral-600">No orders yet. Send a prompt to start.</p>
        )}
        {snapshot.orders.map((order) => (
          <div key={order.id} className="border border-border-subtle p-3">
            <div className="flex items-center justify-between">
              <span className="text-sm text-text-primary truncate">
                {order.title || order.id}
              </span>
              <span
                className={`text-xs uppercase font-body ${
                  order.status === "completed"
                    ? "text-green"
                    : order.status === "failed" || order.status === "failing"
                      ? "text-red"
                      : "text-neutral-500"
                }`}
              >
                {order.status}
              </span>
            </div>
            <div className="mt-1.5 flex gap-1">
              {order.stages.map((stage, i) => (
                <div
                  key={stage.task_key || i}
                  className={`h-1 flex-1 ${
                    stage.status === "completed"
                      ? "bg-green"
                      : stage.status === "active"
                        ? "bg-accent"
                        : stage.status === "failed"
                          ? "bg-red"
                          : "bg-neutral-700"
                  }`}
                />
              ))}
            </div>
          </div>
        ))}
      </div>

      {/* Input area */}
      <div className="p-4 border-t border-border-subtle">
        <div className="flex gap-2">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Send a prompt to the scheduler..."
            rows={2}
            className="flex-1 bg-transparent border border-border-subtle focus:border-accent font-body text-sm text-text-primary p-2 resize-none outline-none placeholder:text-neutral-600"
          />
          <button
            type="button"
            onClick={handleSubmit}
            disabled={isPending || !input.trim()}
            className="self-end bg-accent text-black font-body uppercase text-xs px-4 py-2 disabled:opacity-50"
          >
            SEND
          </button>
        </div>
      </div>
    </div>
  );
}

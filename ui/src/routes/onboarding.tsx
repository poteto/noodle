import { createFileRoute, Link } from "@tanstack/react-router";
import { Check } from "lucide-react";

function OnboardingPage() {
  return (
    <div className="min-h-screen bg-bg-depth text-text-primary flex items-center justify-center p-8">
      <div className="max-w-2xl w-full space-y-12 animate-fade-in">
        {/* Header */}
        <header>
          <div className="flex items-center gap-3 mb-6">
            <div className="logo-mark" />
            <h1 className="font-display text-2xl font-bold tracking-wider text-accent">NOODLE</h1>
          </div>
        </header>

        {/* What is Noodle */}
        <section>
          <h2 className="font-display text-sm font-bold text-accent tracking-wider uppercase mb-3">
            What is Noodle
          </h2>
          <p className="text-text-secondary leading-relaxed">
            Noodle is an open-source AI coding framework built on the kitchen brigade model. It
            orchestrates multiple AI agents through skills — the single extension point — using an
            LLM-powered scheduler to coordinate work across your codebase.
          </p>
        </section>

        {/* How the loop works */}
        <section>
          <h2 className="font-display text-sm font-bold text-accent tracking-wider uppercase mb-4">
            How the loop works
          </h2>
          <ol className="space-y-0">
            {LOOP_STEPS.map((step, i) => (
              <li key={step.label} className="flex items-stretch">
                <div className="flex flex-col items-center mr-4">
                  <div className="w-7 h-7 border-2 border-accent flex items-center justify-center font-mono text-xs font-bold text-accent shrink-0">
                    {i + 1}
                  </div>
                  {i < LOOP_STEPS.length - 1 && <div className="w-px flex-1 bg-border-subtle" />}
                </div>
                <div className="pb-5">
                  <div className="font-mono text-sm font-bold text-text-primary">{step.label}</div>
                  <div className="font-mono text-xs text-text-tertiary mt-0.5">{step.desc}</div>
                </div>
              </li>
            ))}
          </ol>
        </section>

        {/* What you need */}
        <section>
          <h2 className="font-display text-sm font-bold text-accent tracking-wider uppercase mb-4">
            What you need
          </h2>
          <ul className="space-y-3">
            {REQUIREMENTS.map((req) => (
              <li key={req} className="flex items-start gap-3">
                <Check size={14} className="text-accent mt-0.5 shrink-0" />
                <span className="font-mono text-sm text-text-secondary">{req}</span>
              </li>
            ))}
          </ul>
        </section>

        {/* Next steps */}
        <section>
          <h2 className="font-display text-sm font-bold text-accent tracking-wider uppercase mb-4">
            Next steps
          </h2>
          <div className="flex gap-3">
            <Link
              to="/dashboard"
              className="inline-block px-5 py-2.5 bg-accent text-bg-depth font-mono text-sm font-bold tracking-wide hover:brightness-110 transition-[filter] duration-150"
            >
              Go to dashboard
            </Link>
            <Link
              to="/"
              className="inline-block px-5 py-2.5 border border-border-subtle text-text-primary font-mono text-sm font-bold tracking-wide hover:border-border-active hover:bg-bg-surface transition-[border-color,background] duration-150"
            >
              View the live feed
            </Link>
          </div>
        </section>
      </div>
    </div>
  );
}

const LOOP_STEPS = [
  { label: "Schedule", desc: "The scheduler reads your backlog and writes orders" },
  { label: "Execute", desc: "Agents implement changes in isolated worktrees" },
];

const REQUIREMENTS = [
  "Skills — at minimum a schedule skill and an execute skill",
  "A backlog — a todos.md file with tasks for agents to pick up",
  "At least one agent CLI — Claude Code or OpenAI Codex",
];

export const Route = createFileRoute("/onboarding")({
  component: OnboardingPage,
});

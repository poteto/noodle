import { HighlightedCode } from "./CodeHighlight";

interface DiffViewerProps {
  diff: string;
  stat: string;
  isLoading?: boolean;
  error?: string;
}

export function DiffViewer({ diff, stat, isLoading, error }: DiffViewerProps) {
  if (error) {
    return (
      <div className="flex flex-col h-full items-center justify-center p-4">
        <p className="text-red-600 text-sm">{error}</p>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex flex-col h-full items-center justify-center p-4">
        <p className="text-neutral-400 text-sm animate-pulse">Loading diff...</p>
      </div>
    );
  }

  if (!diff && !stat) {
    return (
      <div className="flex flex-col h-full items-center justify-center p-4">
        <p className="text-neutral-400 text-sm">No changes</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      {stat && (
        <div className="px-3 py-2 bg-bg-surface border-b border-border-subtle shrink-0">
          <pre className="whitespace-pre font-mono text-sm text-text-primary leading-relaxed">{stat}</pre>
        </div>
      )}
      <div className="overflow-y-auto flex-1">
        <HighlightedCode code={diff} lang="diff" />
      </div>
    </div>
  );
}

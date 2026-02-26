import { common, createStarryNight } from "@wooorm/starry-night";
import { toHtml } from "hast-util-to-html";
import { useEffect, useState } from "react";

type StarryNight = Awaited<ReturnType<typeof createStarryNight>>;

let starryNightPromise: Promise<StarryNight> | undefined;

function getStarryNight(): Promise<StarryNight> {
  if (!starryNightPromise) {
    starryNightPromise = createStarryNight(common);
  }

  return starryNightPromise;
}

function escapeHtml(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

export function getScopeFromLang(lang: string): string | undefined {
  const normalized = lang.trim().toLowerCase();

  switch (normalized) {
    case "js":
    case "javascript":
      return "source.js";
    case "ts":
    case "typescript":
      return "source.ts";
    case "go":
      return "source.go";
    case "python":
    case "py":
      return "source.python";
    case "bash":
    case "sh":
    case "shell":
      return "source.shell";
    case "json":
      return "source.json";
    case "css":
      return "source.css";
    case "html":
      return "text.html.basic";
    case "rust":
      return "source.rust";
    case "ruby":
    case "rb":
      return "source.ruby";
    case "diff":
      return "source.diff";
    default:
      return undefined;
  }
}

export async function highlightCode(code: string, lang?: string): Promise<string> {
  const scope = lang ? getScopeFromLang(lang) : undefined;
  if (!scope) {
    return escapeHtml(code);
  }

  try {
    const starryNight = await getStarryNight();
    return toHtml(starryNight.highlight(code, scope));
  } catch {
    return escapeHtml(code);
  }
}

export function HighlightedCode({ code, lang }: { code: string; lang?: string }) {
  const [html, setHtml] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setHtml(null);

    void highlightCode(code, lang).then((next) => {
      if (!cancelled) {
        setHtml(next);
      }
    });

    return () => {
      cancelled = true;
    };
  }, [code, lang]);

  if (html === null) {
    return (
      <pre className="my-2 px-3 py-2 bg-[#f5eed8] border-l-2 border-[#e8dfc0] overflow-x-auto text-xs font-mono rounded-sm leading-relaxed">
        {code}
      </pre>
    );
  }

  return (
    <pre
      className="my-2 px-3 py-2 bg-[#f5eed8] border-l-2 border-[#e8dfc0] overflow-x-auto text-xs font-mono rounded-sm leading-relaxed"
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}

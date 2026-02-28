import { useEffect, useState } from "react";
import { Marked } from "marked";
import { highlightCode } from "./CodeHighlight";

const marked = new Marked({
  async: true,
  walkTokens: async (token) => {
    if (token.type === "code") {
      (token as Record<string, unknown>).text = await highlightCode(
        token.text,
        token.lang,
      );
      (token as Record<string, unknown>).escaped = true;
    }
  },
  renderer: {
    code({ text }) {
      return `<pre class="code-block"><code>${text}</code></pre>`;
    },
    codespan({ text }) {
      return `<code class="code-inline">${text}</code>`;
    },
  },
});

export function MarkdownBody({ text }: { text: string }) {
  const [html, setHtml] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (marked.parse(text) as Promise<string>).then((result) => {
      if (!cancelled) setHtml(result);
    });
    return () => {
      cancelled = true;
    };
  }, [text]);

  if (html === null) {
    return <div className="msg-body">{text}</div>;
  }

  return (
    <div
      className="msg-body msg-markdown"
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}

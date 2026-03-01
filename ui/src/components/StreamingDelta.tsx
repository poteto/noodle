import { useEffect, useRef, useState } from "react";
import {
  parser,
  parser_write,
  parser_end,
  default_renderer,
  default_add_token,
  default_end_token,
  default_add_text,
  default_set_attr,
  CODE_BLOCK,
  CODE_FENCE,
  CODE_INLINE,
  LANG,
} from "streaming-markdown";
import type { Default_Renderer_Data, Token, Attr, Parser } from "streaming-markdown";
import { subscribeDelta } from "~/client";
import { highlightCode, getScopeFromLang } from "./CodeHighlight";

function createRenderer(root: HTMLElement) {
  const base = default_renderer(root);

  let currentLang = "";
  let codeNode: HTMLElement | null = null;

  return {
    ...base,
    add_token(data: Default_Renderer_Data, type: Token) {
      default_add_token(data, type);
      const node = data.nodes[data.index];
      if (!node) {
        return;
      }
      if (type === CODE_BLOCK || type === CODE_FENCE) {
        node.parentElement?.classList.add("code-block");
        codeNode = node;
        currentLang = "";
      } else if (type === CODE_INLINE) {
        node.className = "code-inline";
      }
    },
    end_token(data: Default_Renderer_Data) {
      const node = data.nodes[data.index];
      default_end_token(data);
      if (node && node === codeNode) {
        const raw = node.textContent ?? "";
        if (currentLang && getScopeFromLang(currentLang)) {
          void highlightCode(raw, currentLang).then((html) => {
            node.innerHTML = html;
            return html;
          });
        }
        codeNode = null;
        currentLang = "";
      }
    },
    add_text(data: Default_Renderer_Data, text: string) {
      default_add_text(data, text);
    },
    set_attr(data: Default_Renderer_Data, type: Attr, value: string) {
      default_set_attr(data, type, value);
      if (type === LANG) {
        currentLang = value;
      }
    },
  };
}

export function StreamingDelta({ sessionId }: { sessionId: string }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const parserRef = useRef<Parser | null>(null);
  const hasContentRef = useRef(false);
  const [thinking, setThinking] = useState(true);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) {
      return;
    }

    el.innerHTML = "";
    hasContentRef.current = false;
    setThinking(true);

    const renderer = createRenderer(el);
    const p = parser(renderer);
    parserRef.current = p;

    const unsub = subscribeDelta(sessionId, (text) => {
      parser_write(p, text);
      if (!hasContentRef.current && text.trim()) {
        hasContentRef.current = true;
        setThinking(false);
        if (wrapperRef.current) {
          wrapperRef.current.style.display = "";
        }
      }
    });

    return () => {
      unsub();
      parser_end(p);
      parserRef.current = null;
    };
  }, [sessionId]);

  return (
    <>
      {thinking && (
        <div className="thinking-indicator">
          <span className="thinking-dots">
            <span className="thinking-dot" />
            <span className="thinking-dot" />
            <span className="thinking-dot" />
          </span>
          Thinking…
        </div>
      )}
      <div ref={wrapperRef} className="message-row type-system" style={{ display: "none" }}>
        <div className="msg-meta">
          <span className="msg-badge">Think</span>
        </div>
        <div ref={containerRef} className="msg-body msg-markdown" />
      </div>
    </>
  );
}

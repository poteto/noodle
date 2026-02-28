import { useEffect, useRef } from "react";
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
} from "streaming-markdown";
import type { Default_Renderer_Data, Token, Attr, Parser } from "streaming-markdown";
import { subscribeDelta } from "~/client";

function createRenderer(root: HTMLElement) {
  const base = default_renderer(root);

  return {
    ...base,
    add_token(data: Default_Renderer_Data, type: Token) {
      default_add_token(data, type);
      const node = data.nodes[data.index];
      if (!node) {
        return;
      }
      if (type === CODE_BLOCK || type === CODE_FENCE) {
        node.className = "code-block";
      } else if (type === CODE_INLINE) {
        node.className = "code-inline";
      }
    },
    end_token(data: Default_Renderer_Data) {
      default_end_token(data);
    },
    add_text(data: Default_Renderer_Data, text: string) {
      default_add_text(data, text);
    },
    set_attr(data: Default_Renderer_Data, type: Attr, value: string) {
      default_set_attr(data, type, value);
    },
  };
}

export function StreamingDelta({ sessionId }: { sessionId: string }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const parserRef = useRef<Parser | null>(null);
  const hasContentRef = useRef(false);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) {
      return;
    }

    el.innerHTML = "";
    hasContentRef.current = false;

    const renderer = createRenderer(el);
    const p = parser(renderer);
    parserRef.current = p;

    const unsub = subscribeDelta(sessionId, (text) => {
      parser_write(p, text);
      if (!hasContentRef.current && text.trim()) {
        hasContentRef.current = true;
        el.style.display = "";
      }
    });

    return () => {
      unsub();
      parser_end(p);
      parserRef.current = null;
    };
  }, [sessionId]);

  return (
    <div className="message-row type-system">
      <div className="msg-avatar">TH</div>
      <div>
        <div className="msg-meta">
          <span className="msg-badge">Think</span>
        </div>
        <div ref={containerRef} className="msg-body msg-markdown" style={{ display: "none" }} />
      </div>
    </div>
  );
}

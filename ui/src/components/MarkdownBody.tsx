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
  LANG,
} from "streaming-markdown";
import type { Default_Renderer_Data, Token, Attr } from "streaming-markdown";
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
        // default_add_token creates <pre><code>; node is the <code>.
        // Put "code-block" on <pre> — LANG attr overwrites <code>'s class.
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

export function MarkdownBody({ text }: { text: string }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const renderedTextRef = useRef("");

  useEffect(() => {
    const el = containerRef.current;
    if (!el || text === renderedTextRef.current) {
      return;
    }

    el.innerHTML = "";
    renderedTextRef.current = text;

    const renderer = createRenderer(el);
    const p = parser(renderer);
    parser_write(p, text);
    parser_end(p);
  }, [text]);

  return <div ref={containerRef} className="msg-body msg-markdown" />;
}

import { defineConfig } from "vitepress";
import { buildEndGenerateOpenGraphImages } from "./plugins/og-image.mjs";

export default defineConfig({
  base: "/noodle/",
  title: "Noodle",
  description: "Skills that run themselves. Orchestrate agents using skills.",

  markdown: {
    theme: "min-dark",
  },
  head: [
    ["link", { rel: "icon", type: "image/svg+xml", href: "/noodle/favicon.svg" }],
    [
      "link",
      {
        rel: "stylesheet",
        href: "https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&display=swap",
      },
    ],
  ],

  transformHead({ pageData, siteData }) {
    // Add OG tags for pages the plugin doesn't cover (homepage, 404, cookbook index)
    if (pageData.relativePath === "index.md") {
      return [
        ["meta", { property: "og:title", content: siteData.title }],
        ["meta", { property: "og:description", content: siteData.description }],
        ["meta", { property: "og:url", content: "https://poteto.github.io/noodle/" }],
        ["meta", { property: "og:site_name", content: siteData.title }],
        ["meta", { property: "og:type", content: "website" }],
        ["meta", { property: "og:image", content: "https://poteto.github.io/noodle/og-introduction.png" }],
        ["meta", { property: "og:image:width", content: "1200" }],
        ["meta", { property: "og:image:height", content: "630" }],
        ["meta", { name: "twitter:card", content: "summary_large_image" }],
        ["meta", { name: "twitter:image", content: "https://poteto.github.io/noodle/og-introduction.png" }],
      ];
    }
  },

  appearance: false,

  buildEnd: buildEndGenerateOpenGraphImages({
    baseUrl: "https://poteto.github.io/noodle",
    category: {
      byCustomGetter: (page) => {
        const dir = page.sourceFilePath.split("/")[1];
        if (dir === "concepts") return "Concepts";
        if (dir === "reference") return "Reference";
        if (dir === "cookbook") return "Cookbook";
        return "Guide";
      },
      fallbackWithFrontmatter: false,
    },
  }),

  themeConfig: {
    nav: [
      ...(process.env.NOODLE_VERSION
        ? [{ text: process.env.NOODLE_VERSION, link: `https://github.com/poteto/noodle/releases/tag/${process.env.NOODLE_VERSION}` }]
        : []),
      { text: "GitHub", link: "https://github.com/poteto/noodle" },
    ],

    sidebar: [
      {
        text: "Introduction",
        link: "/introduction",
      },
      {
        text: "Getting Started",
        link: "/getting-started",
      },
      {
        text: "Thinking in Noodle",
        link: "/thinking-in-noodle",
      },
      {
        text: "Concepts",
        items: [
          { text: "Skills", link: "/concepts/skills" },
          { text: "Scheduling", link: "/concepts/scheduling" },
          { text: "Events", link: "/concepts/events" },
          { text: "Adapters", link: "/concepts/adapters" },
          { text: "Modes", link: "/concepts/modes" },
          { text: "Runtimes", link: "/concepts/runtimes" },
          { text: "Self-Learning (Optional)", link: "/concepts/brain" },
        ],
      },
      {
        text: "Reference",
        items: [
          { text: "CLI", link: "/reference/cli" },
          { text: "Configuration", link: "/reference/configuration" },
          { text: "Glossary", link: "/reference/glossary" },
          { text: "FAQ", link: "/reference/faq" },
        ],
      },
      {
        text: "Cookbook",
        items: [
          {
            text: "Minimal Noodle Loop",
            link: "/cookbook/minimal-noodle-loop",
          },
          {
            text: "Multi-Stage Pipeline",
            link: "/cookbook/multi-stage-pipeline",
          },
          { text: "Self-Learning", link: "/cookbook/self-learning" },
          { text: "Model Routing", link: "/cookbook/model-routing" },
        ],
      },
      {
        text: "Community",
        items: [
          {
            text: "Discord",
            link: "https://discord.gg/RmJqTgkMz9",
          },
          {
            text: "File an Issue",
            link: "https://github.com/poteto/noodle/issues",
          },
        ],
      },
    ],

    outline: {
      level: [2, 3],
      label: "On this page",
    },

    search: {
      provider: "local",
    },
  },
});

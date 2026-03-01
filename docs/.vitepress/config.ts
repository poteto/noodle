import { defineConfig } from "vitepress";

export default defineConfig({
  base: "/noodle/",
  title: "Noodle",
  description:
    "Skill-based agent orchestration. Built in Go.",

  markdown: {
    theme: "min-dark",
  },
  head: [
    [
      "link",
      {
        rel: "stylesheet",
        href: "https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&display=swap",
      },
    ],
  ],

  appearance: false,

  themeConfig: {
    nav: [{ text: "GitHub", link: "https://github.com/poteto/noodle" }],

    sidebar: [
      {
        text: "Getting Started",
        link: "/getting-started",
      },
      {
        text: "Concepts",
        items: [
          { text: "Skills", link: "/concepts/skills" },
          { text: "Scheduling", link: "/concepts/scheduling" },
          { text: "Brain", link: "/concepts/brain" },
          { text: "Modes", link: "/concepts/modes" },
          { text: "Philosophy", link: "/concepts/philosophy" },
        ],
      },
      {
        text: "Reference",
        items: [
          { text: "Configuration", link: "/reference/configuration" },
        ],
      },
      {
        text: "Examples",
        link: "/examples",
      },
      {
        text: "Contributing",
        items: [
          {
            text: "Failure Message Policy",
            link: "/contributing/failure-message-policy",
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

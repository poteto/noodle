import { defineConfig } from "vitepress";

export default defineConfig({
  base: "/noodle/",
  title: "Noodle",
  description: "Skill-based agent orchestration. Built in Go.",

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
        text: "Vision",
        link: "/vision",
      },
      {
        text: "Getting Started",
        link: "/getting-started",
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
          { text: "Glossary", link: "/glossary" },
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

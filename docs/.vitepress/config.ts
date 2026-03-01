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
        text: "Why Noodle",
        link: "/why-noodle",
      },
      {
        text: "Getting Started",
        link: "/getting-started",
      },
      {
        text: "Install (for Agents)",
        link: "/install",
      },
      {
        text: "Concepts",
        items: [
          { text: "Skills", link: "/concepts/skills" },
          { text: "Scheduling", link: "/concepts/scheduling" },
          { text: "Brain", link: "/concepts/brain" },
          { text: "Modes", link: "/concepts/modes" },
          { text: "Runtimes", link: "/concepts/runtimes" },
        ],
      },
      {
        text: "Reference",
        items: [
          { text: "CLI", link: "/reference/cli" },
          { text: "Configuration", link: "/reference/configuration" },
          { text: "Skill Frontmatter", link: "/reference/skill-frontmatter" },
        ],
      },
      {
        text: "Cookbook",
        collapsed: false,
        items: [
          { text: "Minimal Loop", link: "/cookbook/minimal-loop" },
          { text: "Multi-Stage Pipeline", link: "/cookbook/multi-stage-pipeline" },
          { text: "Self-Learning", link: "/cookbook/self-learning" },
          { text: "Model Routing", link: "/cookbook/model-routing" },
        ],
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

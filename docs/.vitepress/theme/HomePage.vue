<script setup>
import { withBase } from "vitepress";
</script>

<template>
  <div class="home">
    <!-- Header -->
    <header class="header">
      <div class="header-inner">
        <div class="header-left">
          <a :href="withBase('/')" class="brand">NOODLE</a>
          <nav class="header-nav">
            <a :href="withBase('/getting-started')" class="nav-link">Docs</a>
            <a :href="withBase('/cookbook/minimal-loop')" class="nav-link">Cookbook</a>
          </nav>
        </div>
        <div class="header-right">
          <div class="divider-v"></div>
          <a
            href="https://github.com/poteto/noodle"
            class="github-link"
            target="_blank"
            rel="noopener"
          >
            GitHub
            <svg
              width="12"
              height="12"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M7 7h10v10" />
              <path d="M7 17 17 7" />
            </svg>
          </a>
        </div>
      </div>
    </header>

    <!-- Main -->
    <main class="main">
      <div class="container">
        <!-- Version badge -->
        <div class="hero-line-1 badge-row">
          <span class="tag">v0.1.0</span>
          <span class="badge-label">— MIT License</span>
        </div>

        <!-- Hero -->
        <section class="hero">
          <h1 class="hero-line-2 hero-title">NOODLE</h1>
          <h2 class="hero-line-3 hero-subtitle">
            <span class="block">Skill-based</span>
            <span class="cursor">agent orchestration</span>
          </h2>

          <p class="hero-sub hero-desc">
            Build your own agent orchestration workflow.<br />
            All you need are skills.
          </p>

          <div class="hero-cta cta-row">
            <a :href="withBase('/getting-started')" class="btn-primary">
              Get Started
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <path d="M5 12h14" />
                <path d="m12 5 7 7-7 7" />
              </svg>
            </a>
            <a
              href="https://github.com/poteto/noodle"
              class="btn-secondary"
              target="_blank"
              rel="noopener"
            >
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <path
                  d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4"
                />
                <path d="M9 18c-4.51 2-5-2-7-2" />
              </svg>
              View on GitHub
            </a>
          </div>
        </section>

        <hr class="divider-line" />

        <!-- Feature cards -->
        <section class="cards-row cards-section">
          <p class="section-label">Core Primitives</p>
          <div class="cards-grid">
            <div class="feature-card">
              <div class="card-header">
                <div class="card-icon">
                  <svg
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  >
                    <path
                      d="M4 14a1 1 0 0 1-.78-1.63l9.9-10.2a.5.5 0 0 1 .86.46l-1.92 6.02A1 1 0 0 0 13 10h7a1 1 0 0 1 .78 1.63l-9.9 10.2a.5.5 0 0 1-.86-.46l1.92-6.02A1 1 0 0 0 11 14z"
                    />
                  </svg>
                </div>
                <h3 class="card-title">Skills</h3>
              </div>
              <p class="card-desc">
                Composable and reusable. Build agent orchestration exactly how
                you want it.
              </p>
              <a :href="withBase('/concepts/skills')" class="card-link">
                Learn more
                <svg
                  width="12"
                  height="12"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                >
                  <path d="M5 12h14" />
                  <path d="m12 5 7 7-7 7" />
                </svg>
              </a>
            </div>

            <div class="feature-card">
              <div class="card-header">
                <div class="card-icon">
                  <svg
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  >
                    <line x1="6" x2="6" y1="3" y2="15" />
                    <circle cx="18" cy="6" r="3" />
                    <circle cx="6" cy="18" r="3" />
                    <path d="M18 9a9 9 0 0 1-9 9" />
                  </svg>
                </div>
                <h3 class="card-title">Scheduling</h3>
              </div>
              <p class="card-desc">
                Agent-powered task scheduling that routes work to the right
                model at the right time, with full observability.
              </p>
              <a :href="withBase('/concepts/scheduling')" class="card-link">
                Learn more
                <svg
                  width="12"
                  height="12"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                >
                  <path d="M5 12h14" />
                  <path d="m12 5 7 7-7 7" />
                </svg>
              </a>
            </div>

            <div class="feature-card">
              <div class="card-header">
                <div class="card-icon">
                  <svg
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  >
                    <ellipse cx="12" cy="5" rx="9" ry="3" />
                    <path d="M3 5V19A9 3 0 0 0 21 19V5" />
                    <path d="M3 12A9 3 0 0 0 21 12" />
                  </svg>
                </div>
                <h3 class="card-title">Brain</h3>
              </div>
              <p class="card-desc">
                Optional persistent memory across sessions. An Obsidian vault
                that grows and evolves with your project over time.
              </p>
              <a :href="withBase('/concepts/brain')" class="card-link">
                Learn more
                <svg
                  width="12"
                  height="12"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                >
                  <path d="M5 12h14" />
                  <path d="m12 5 7 7-7 7" />
                </svg>
              </a>
            </div>

            <div class="feature-card">
              <div class="card-header">
                <div class="card-icon">
                  <svg
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  >
                    <line x1="4" x2="4" y1="21" y2="14" />
                    <line x1="4" x2="4" y1="10" y2="3" />
                    <line x1="12" x2="12" y1="21" y2="12" />
                    <line x1="12" x2="12" y1="8" y2="3" />
                    <line x1="20" x2="20" y1="21" y2="16" />
                    <line x1="20" x2="20" y1="12" y2="3" />
                    <line x1="2" x2="6" y1="14" y2="14" />
                    <line x1="10" x2="14" y1="8" y2="8" />
                    <line x1="18" x2="22" y1="16" y2="16" />
                  </svg>
                </div>
                <h3 class="card-title">Modes</h3>
              </div>
              <p class="card-desc">
                Auto, supervised, and manual. Fine-tune how autonomous your
                agents are at each step.
              </p>
              <a :href="withBase('/concepts/modes')" class="card-link">
                Learn more
                <svg
                  width="12"
                  height="12"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                >
                  <path d="M5 12h14" />
                  <path d="m12 5 7 7-7 7" />
                </svg>
              </a>
            </div>
          </div>
        </section>

        <!-- Quick-start strip -->
        <div class="quickstart">
          <div class="quickstart-left">
            <p class="quickstart-label">Quick Install</p>
            <code class="quickstart-cmd"
              ><span class="cmd-prompt">$</span> brew install
              poteto/tap/noodle</code
            >
          </div>
          <div class="quickstart-links">
            <a :href="withBase('/getting-started')" class="quickstart-link"
              >Documentation →</a
            >
            <a :href="withBase('/reference/configuration')" class="quickstart-link"
              >Configuration →</a
            >
            <a :href="withBase('/cookbook/minimal-loop')" class="quickstart-link"
              >Cookbook →</a
            >
          </div>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <footer class="footer">
      <div class="footer-inner">
        <span class="footer-text">LAUREN TAN © 2026</span>
        <span class="footer-text">MIT License</span>
      </div>
    </footer>
  </div>
</template>

<style scoped>
/* ============================================================
   Base
   ============================================================ */

.home {
  font-family: "JetBrains Mono", monospace;
  background-color: #080808;
  color: #ffffff;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  width: 100%;
  overflow-x: hidden;
}

.home ::selection {
  background: #facc15;
  color: #000;
}

/* ============================================================
   Header
   ============================================================ */

.header {
  width: 100%;
  border-bottom: 1px solid #161616;
  background: rgba(8, 8, 8, 0.95);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  position: sticky;
  top: 0;
  z-index: 50;
}

.header-inner {
  max-width: 1360px;
  margin: 0 auto;
  padding: 0 32px;
  height: 52px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 40px;
}

.brand {
  color: #fff;
  font-weight: 700;
  letter-spacing: 0.2em;
  font-size: 14px;
  text-decoration: none;
  transition: color 0.2s;
}

.brand:hover {
  color: #facc15;
}

.header-nav {
  display: none;
  align-items: center;
  gap: 24px;
}

@media (min-width: 768px) {
  .header-nav {
    display: flex;
  }
}

.nav-link {
  font-size: 13px;
  color: #8a8a8a;
  text-decoration: none;
  transition: color 0.2s;
  letter-spacing: 0.05em;
}

.nav-link:hover {
  color: #d4d4d4;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 20px;
}

.divider-v {
  height: 16px;
  width: 1px;
  background: #1e1e1e;
}

.github-link {
  font-size: 13px;
  color: #8a8a8a;
  text-decoration: none;
  display: flex;
  align-items: center;
  gap: 4px;
  transition: color 0.2s;
}

.github-link:hover {
  color: #fff;
}

/* ============================================================
   Main
   ============================================================ */

.main {
  flex-grow: 1;
  width: 100%;
  padding: 80px 32px 128px;
  background-image: radial-gradient(#222222 1px, transparent 0);
  background-size: 20px 20px;
}

.container {
  max-width: 1360px;
  margin: 0 auto;
}

/* ============================================================
   Version badge
   ============================================================ */

.badge-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 40px;
}

.tag {
  display: inline-block;
  padding: 2px 8px;
  background: rgba(250, 204, 21, 0.08);
  border: 1px solid rgba(250, 204, 21, 0.2);
  border-radius: 3px;
  color: #facc15;
  font-size: 10px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  font-weight: 600;
}

.badge-label {
  font-size: 11px;
  color: #6b6b6b;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

/* ============================================================
   Hero
   ============================================================ */

.hero {
  margin-bottom: 80px;
}

.hero-title {
  font-weight: 700;
  line-height: 1;
  margin-bottom: 8px;
  letter-spacing: -0.02em;
  font-size: clamp(56px, 8vw, 96px);
  color: #fff;
}

.hero-subtitle {
  font-weight: 700;
  line-height: 1;
  margin-bottom: 40px;
  letter-spacing: -0.02em;
  font-size: clamp(56px, 8vw, 96px);
  color: #444444;
}

.block {
  display: block;
}

.cursor::after {
  content: "_";
  animation: blink 1.2s step-end infinite;
  color: #facc15;
  font-weight: 700;
}

.hero-desc {
  font-size: 1.1em;
  color: #9a9a9a;
  max-width: 560px;
  margin-bottom: 40px;
  line-height: 1.8;
}

.cta-row {
  display: flex;
  flex-direction: row;
  gap: 12px;
}

.btn-primary {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 24px;
  background: #facc15;
  color: #000;
  font-weight: 700;
  font-size: 14px;
  letter-spacing: 0.025em;
  border-radius: 2px;
  text-decoration: none;
  transition: background 0.2s;
}

.btn-primary:hover {
  background: #fde047;
}

.btn-secondary {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 24px;
  background: transparent;
  border: 1px solid #222;
  color: #a3a3a3;
  font-weight: 500;
  font-size: 14px;
  letter-spacing: 0.025em;
  border-radius: 2px;
  text-decoration: none;
  transition: border-color 0.2s, color 0.2s;
}

.btn-secondary:hover {
  border-color: #333;
  color: #fff;
}

/* ============================================================
   Divider
   ============================================================ */

.divider-line {
  border: none;
  border-top: 1px solid #161616;
  margin: 0 0 64px;
}

/* ============================================================
   Feature cards
   ============================================================ */

.section-label {
  font-size: 11px;
  color: #555555;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  margin-bottom: 32px;
}

.cards-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1px;
  background: #161616;
}

@media (min-width: 768px) {
  .cards-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (min-width: 1024px) {
  .cards-grid {
    grid-template-columns: repeat(4, 1fr);
  }
}

.feature-card {
  position: relative;
  overflow: hidden;
  background: #080808;
  padding: 32px;
  border: 1px solid transparent;
  transition: border-color 0.2s, background 0.2s;
  display: flex;
  flex-direction: column;
}

.feature-card::before {
  content: "";
  position: absolute;
  inset: 0;
  background: linear-gradient(
    135deg,
    rgba(250, 204, 21, 0.04) 0%,
    transparent 60%
  );
  opacity: 0;
  transition: opacity 0.3s;
}

.feature-card:hover::before {
  opacity: 1;
}

.feature-card:hover {
  border-color: #2a2a2a;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 20px;
}

.card-icon {
  width: 28px;
  height: 28px;
  border-radius: 2px;
  background: #111;
  border: 1px solid #1e1e1e;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #facc15;
}

.card-title {
  color: #fff;
  font-weight: 600;
  font-size: 14px;
  letter-spacing: 0.025em;
}

.card-desc {
  color: #6b6b6b;
  font-size: 1.1em;
  line-height: 1.9;
  position: relative;
  z-index: 1;
}

.card-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: auto;
  padding-top: 24px;
  font-size: 1em;
  color: rgba(250, 204, 21, 0.6);
  text-decoration: none;
  transition: color 0.2s;
  letter-spacing: 0.05em;
  position: relative;
  z-index: 1;
}

.card-link:hover {
  color: #facc15;
}

/* ============================================================
   Quick-start strip
   ============================================================ */

.quickstart {
  margin-top: 64px;
  padding: 24px;
  background: #0a0a0a;
  border: 1px solid #161616;
  border-radius: 2px;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
}

@media (min-width: 768px) {
  .quickstart {
    flex-direction: row;
    align-items: center;
  }
}

.quickstart-label {
  font-size: 10px;
  color: #6b6b6b;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  margin-bottom: 4px;
}

.quickstart-cmd {
  font-size: 14px;
  color: #d4d4d4;
}

.cmd-prompt {
  color: #525252;
}

.quickstart-links {
  display: flex;
  align-items: center;
  gap: 16px;
}

.quickstart-link {
  font-size: 13px;
  color: #8a8a8a;
  text-decoration: none;
  transition: color 0.2s;
  letter-spacing: 0.05em;
}

.quickstart-link:hover {
  color: #fff;
}

/* ============================================================
   Footer
   ============================================================ */

.footer {
  border-top: 1px solid #161616;
  padding: 24px 32px;
}

.footer-inner {
  max-width: 1360px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.footer-text {
  font-size: 12px;
  color: #555555;
  letter-spacing: 0.1em;
}

/* ============================================================
   Animations
   ============================================================ */

@keyframes blink {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0;
  }
}

@keyframes fadeUp {
  from {
    opacity: 0;
    transform: translateY(16px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.hero-line-1 {
  animation: fadeUp 0.6s ease forwards;
}
.hero-line-2 {
  animation: fadeUp 0.6s 0.1s ease forwards;
  opacity: 0;
}
.hero-line-3 {
  animation: fadeUp 0.6s 0.2s ease forwards;
  opacity: 0;
}
.hero-sub {
  animation: fadeUp 0.6s 0.35s ease forwards;
  opacity: 0;
}
.hero-cta {
  animation: fadeUp 0.6s 0.5s ease forwards;
  opacity: 0;
}
.cards-row {
  animation: fadeUp 0.6s 0.65s ease forwards;
  opacity: 0;
}
</style>

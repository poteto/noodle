<script setup>
import { withBase } from "vitepress";
</script>

<template>
  <div class="not-found">
    <div class="container">
      <p class="label">ERROR</p>
      <h1 class="title">404</h1>
      <h2 class="subtitle">Skill Issue</h2>
      <p class="caption">
        We looked everywhere. Checked the backlog, read the mise, even asked the
        scheduler. This page doesn't exist.
      </p>
      <div class="actions">
        <a :href="withBase('/')" class="btn-primary">
          Back to safety
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
        <a :href="withBase('/getting-started')" class="btn-secondary">
          Read the docs
        </a>
      </div>
    </div>
  </div>
</template>

<style scoped>
.not-found {
  font-family: "JetBrains Mono", monospace;
  background-color: #080808;
  background-image: radial-gradient(#222222 1px, transparent 0);
  background-size: 20px 20px;
  color: #ffffff;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px;
}

.not-found ::selection {
  background: #facc15;
  color: #000;
}

.container {
  max-width: 560px;
  text-align: left;
}

.label {
  font-size: 10px;
  color: #555555;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  margin-bottom: 24px;
}

.title {
  font-weight: 700;
  font-size: clamp(72px, 12vw, 128px);
  line-height: 1;
  letter-spacing: -0.02em;
  color: #facc15;
  margin-bottom: 0;
}

.subtitle {
  font-weight: 700;
  font-size: clamp(32px, 5vw, 48px);
  line-height: 1.1;
  letter-spacing: -0.02em;
  color: #444444;
  margin-bottom: 32px;
}

.caption {
  font-size: 14px;
  color: #6b6b6b;
  line-height: 1.8;
  margin-bottom: 40px;
}

.actions {
  display: flex;
  gap: 12px;
}

.btn-primary {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 24px;
  background: #facc15;
  color: #000;
  font-family: "JetBrains Mono", monospace;
  font-weight: 700;
  font-size: 14px;
  letter-spacing: 0.025em;
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
  font-family: "JetBrains Mono", monospace;
  font-weight: 500;
  font-size: 14px;
  letter-spacing: 0.025em;
  text-decoration: none;
  transition: border-color 0.2s, color 0.2s;
}

.btn-secondary:hover {
  border-color: #333;
  color: #fff;
}
</style>

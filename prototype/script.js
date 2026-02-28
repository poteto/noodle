// ============================================================
// NOODLE — Agent Conversations UI Prototype
// Monochromatic dark, three-column orchestrator
// ============================================================

const CHANNELS = {
  schedule: {
    name: "schedule",
    title: "Scheduler",
    taskKey: "schedule",
    model: "claude opus 4.6",
    cost: 0.53,
    ctx: 12,
    status: "working",
    avatar: "S",
    isManager: true,
    host: "local",
    statusText: "Scheduling next batch...",
    stages: [
      { name: "schedule", status: "active" },
    ],
    files: [],
    messages: [
      { type: "system", body: "Session started · provider: claude" },
      {
        from: "schedule",
        avatar: "S",
        isManager: true,
        label: "SCHEDULER",
        time: "2:58 PM",
        badge: "PLANNING",
        body: "Reading backlog from .noodle/mise.json to determine what to schedule next. Let me check the current orders state and backlog priorities.",
      },
      {
        from: "schedule",
        avatar: "S",
        isManager: true,
        type: "tool",
        label: "SCHEDULER",
        time: "2:58 PM",
        toolName: "Read",
        toolTarget: ".noodle/mise.json",
      },
      {
        from: "schedule",
        avatar: "S",
        isManager: true,
        type: "tool",
        label: "SCHEDULER",
        time: "2:58 PM",
        toolName: "Read",
        toolTarget: ".noodle/orders.json",
      },
      {
        from: "schedule",
        avatar: "S",
        isManager: true,
        label: "SCHEDULER",
        time: "2:59 PM",
        body: 'The backlog has 3 items. I should schedule the highest priority items that aren\'t already being worked on.\n\nWriting schedule to <code>orders-next.json</code>...',
      },
      {
        from: "schedule",
        avatar: "S",
        isManager: true,
        type: "tool",
        label: "SCHEDULER",
        time: "2:59 PM",
        toolName: "Skill",
        toolTarget: "schedule — writing orders-next.json",
      },
      {
        from: "schedule",
        avatar: "S",
        isManager: true,
        label: "SCHEDULER",
        time: "3:00 PM",
        badge: "RESULT",
        body: 'Schedule written to <code>orders-next.json</code>. 2 items scheduled:\n\n1. <code>Add dark mode toggle</code> — execute → quality → reflect\n2. <code>Fix auth redirect loop</code> — execute → quality\n\nThey can be scheduled in parallel since they don\'t share files.',
      },
      { type: "cost", body: "$0.53 · 3k in / 1.2k out" },
      { type: "typing" },
    ],
  },
  "golden-thyme": {
    name: "golden-thyme",
    title: "Fix auth redirect loop",
    taskKey: "execute",
    model: "claude opus 4.6",
    cost: 0.42,
    ctx: 28,
    status: "idle",
    avatar: "GT",
    isManager: false,
    host: "sprite",
    statusText: "Idle",
    stages: [
      { name: "execute", status: "done" },
      { name: "quality", status: "pending" },
    ],
    files: [
      { name: "src/auth/middleware.go", action: "edit" },
      { name: "src/auth/redirect.go", action: "edit" },
      { name: "src/auth/middleware_test.go", action: "write" },
    ],
    messages: [
      { type: "system", body: "Session started · provider: claude" },
      {
        from: "user",
        avatar: "→",
        isUser: true,
        label: "PROMPT",
        time: "2:42 PM",
        body: "[order:task-1772233154331] Work backlog item task-1772233154331\nTask: Fix auth redirect loop — unauthenticated users get stuck in infinite redirect between /login and /callback",
      },
      {
        from: "golden-thyme",
        avatar: "GT",
        label: "GOLDEN-THYME",
        time: "2:42 PM",
        body: "Let me trace the auth flow to understand where the redirect loop occurs.",
      },
      {
        from: "golden-thyme",
        avatar: "GT",
        type: "tool",
        label: "GOLDEN-THYME",
        time: "2:42 PM",
        toolName: "Read",
        toolTarget: "src/auth/middleware.go",
      },
      {
        from: "golden-thyme",
        avatar: "GT",
        type: "tool",
        label: "GOLDEN-THYME",
        time: "2:43 PM",
        toolName: "Read",
        toolTarget: "src/auth/redirect.go",
      },
      {
        from: "golden-thyme",
        avatar: "GT",
        label: "GOLDEN-THYME",
        time: "2:43 PM",
        body: "Found the bug. The middleware checks <code>session.IsValid()</code> but the OAuth callback handler doesn't set the session until after the redirect completes. So <code>/callback</code> itself triggers the auth middleware, which redirects back to <code>/login</code>.\n\nThe fix is to exempt <code>/callback</code> from the auth middleware, same as <code>/login</code> is already exempted.",
      },
      {
        from: "golden-thyme",
        avatar: "GT",
        type: "tool",
        label: "GOLDEN-THYME",
        time: "2:44 PM",
        toolName: "Edit",
        toolTarget: "src/auth/middleware.go — add /callback to skip list",
      },
      {
        from: "golden-thyme",
        avatar: "GT",
        type: "tool",
        label: "GOLDEN-THYME",
        time: "2:44 PM",
        toolName: "Edit",
        toolTarget: "src/auth/redirect.go — add cycle detection guard",
      },
      {
        from: "golden-thyme",
        avatar: "GT",
        label: "GOLDEN-THYME",
        time: "2:45 PM",
        body: "I also added a redirect cycle detection guard that caps consecutive redirects at 3 and returns a 400 error instead. This prevents any future redirect loops regardless of route config.",
      },
      {
        from: "golden-thyme",
        avatar: "GT",
        type: "tool",
        label: "GOLDEN-THYME",
        time: "2:45 PM",
        toolName: "Write",
        toolTarget: "src/auth/middleware_test.go",
      },
      {
        from: "golden-thyme",
        avatar: "GT",
        type: "tool",
        label: "GOLDEN-THYME",
        time: "2:46 PM",
        toolName: "Bash",
        toolTarget: "go test ./src/auth/... -run TestRedirectLoop -v",
      },
      {
        from: "golden-thyme",
        avatar: "GT",
        label: "GOLDEN-THYME",
        time: "2:46 PM",
        badge: "RESULT",
        body: "All tests pass including the new <code>TestRedirectLoop</code> case. The fix:\n\n1. Exempted <code>/callback</code> from auth middleware\n2. Added redirect cycle detection (max 3 hops)\n3. Added regression test covering the loop scenario",
      },
      { type: "cost", body: "$0.42 · 8k in / 2.1k out" },
      { type: "idle" },
    ],
  },
  "copper-sage": {
    name: "copper-sage",
    title: "Add dark mode toggle",
    taskKey: "execute",
    model: "claude opus 4.6",
    cost: 0.89,
    ctx: 34,
    status: "working",
    avatar: "CS",
    isManager: false,
    host: "local",
    statusText: "Editing SettingsPage.tsx",
    stages: [
      { name: "execute", status: "active" },
      { name: "quality", status: "pending" },
    ],
    files: [
      { name: "src/theme/tokens.ts", action: "edit" },
      { name: "src/components/ThemeToggle.tsx", action: "write" },
      { name: "src/components/SettingsPage.tsx", action: "edit" },
    ],
    messages: [
      { type: "system", body: "Session started · provider: claude" },
      {
        from: "user",
        avatar: "→",
        isUser: true,
        label: "PROMPT",
        time: "2:59 PM",
        body: "[order:task-1772233154332] Work backlog item task-1772233154332\nTask: Add dark mode toggle to the settings page",
      },
      {
        from: "copper-sage",
        avatar: "CS",
        label: "COPPER-SAGE",
        time: "2:59 PM",
        body: "Let me explore the current settings page and understand the theming system before making changes.",
      },
      {
        from: "copper-sage",
        avatar: "CS",
        type: "tool",
        label: "COPPER-SAGE",
        time: "2:59 PM",
        toolName: "Glob",
        toolTarget: "src/components/Settings*.tsx",
      },
      {
        from: "copper-sage",
        avatar: "CS",
        type: "tool",
        label: "COPPER-SAGE",
        time: "3:00 PM",
        toolName: "Read",
        toolTarget: "src/components/SettingsPage.tsx",
      },
      {
        from: "copper-sage",
        avatar: "CS",
        type: "tool",
        label: "COPPER-SAGE",
        time: "3:00 PM",
        toolName: "Read",
        toolTarget: "src/theme/tokens.ts",
      },
      {
        from: "copper-sage",
        avatar: "CS",
        label: "COPPER-SAGE",
        time: "3:00 PM",
        body: "The theme system uses CSS custom properties. I need to:\n1. Add a <code>dark</code> theme variant to <code>tokens.ts</code>\n2. Create a toggle component in the settings page\n3. Persist the preference to localStorage",
      },
      {
        from: "copper-sage",
        avatar: "CS",
        type: "tool",
        label: "COPPER-SAGE",
        time: "3:01 PM",
        toolName: "Edit",
        toolTarget: "src/theme/tokens.ts — adding dark theme variant",
      },
      {
        from: "copper-sage",
        avatar: "CS",
        type: "tool",
        label: "COPPER-SAGE",
        time: "3:01 PM",
        toolName: "Write",
        toolTarget: "src/components/ThemeToggle.tsx",
      },
      {
        from: "copper-sage",
        avatar: "CS",
        type: "tool",
        label: "COPPER-SAGE",
        time: "3:02 PM",
        toolName: "Edit",
        toolTarget: "src/components/SettingsPage.tsx — adding ThemeToggle",
      },
      {
        from: "copper-sage",
        avatar: "CS",
        label: "COPPER-SAGE",
        time: "3:02 PM",
        body: "Running the test suite to make sure the toggle works and doesn't break existing tests...",
      },
      {
        from: "copper-sage",
        avatar: "CS",
        type: "tool",
        label: "COPPER-SAGE",
        time: "3:02 PM",
        toolName: "Bash",
        toolTarget: "pnpm test -- --run",
      },
      { type: "typing" },
    ],
  },
  "silver-oak": {
    name: "silver-oak",
    title: "Update README links",
    taskKey: "execute",
    model: "claude haiku 4.5",
    cost: 0.12,
    ctx: 0,
    status: "done",
    avatar: "SO",
    isManager: false,
    host: "sprite",
    statusText: "Completed",
    stages: [
      { name: "execute", status: "done" },
      { name: "quality", status: "done" },
    ],
    files: [
      { name: "README.md", action: "edit" },
      { name: "packages/core/README.md", action: "edit" },
    ],
    messages: [
      { type: "system", body: "Session started · provider: claude" },
      {
        from: "user",
        avatar: "→",
        isUser: true,
        label: "PROMPT",
        time: "2:45 PM",
        body: "[order:task-1772233154330] Work backlog item task-1772233154330\nTask: Update README links to point to new docs site",
      },
      {
        from: "silver-oak",
        avatar: "SO",
        label: "SILVER-OAK",
        time: "2:45 PM",
        body: "Simple find-and-replace of the old docs URL with the new one across all README files.",
      },
      {
        from: "silver-oak",
        avatar: "SO",
        type: "tool",
        label: "SILVER-OAK",
        time: "2:45 PM",
        toolName: "Grep",
        toolTarget: "docs.old-site.com — 4 matches",
      },
      {
        from: "silver-oak",
        avatar: "SO",
        type: "tool",
        label: "SILVER-OAK",
        time: "2:46 PM",
        toolName: "Edit",
        toolTarget: "README.md — updated 2 links",
      },
      {
        from: "silver-oak",
        avatar: "SO",
        type: "tool",
        label: "SILVER-OAK",
        time: "2:46 PM",
        toolName: "Edit",
        toolTarget: "packages/core/README.md — updated 2 links",
      },
      {
        from: "silver-oak",
        avatar: "SO",
        label: "SILVER-OAK",
        time: "2:46 PM",
        body: "All 4 links updated. Committing.",
      },
      {
        from: "silver-oak",
        avatar: "SO",
        type: "tool",
        label: "SILVER-OAK",
        time: "2:46 PM",
        toolName: "Bash",
        toolTarget: 'git commit -m "docs: Update README links to new docs site"',
      },
      { type: "cost", body: "$0.12 · 8k in / 340 out" },
      { type: "system", body: "Completed · merged to main" },
    ],
  },
};

let activeChannel = "schedule";

// --- Render sidebar ---

function renderSidebar() {
  const managerList = document.getElementById("agent-list-manager");
  const treeContainer = document.getElementById("agent-tree");

  managerList.innerHTML = "";
  treeContainer.innerHTML = "";

  var orders = [];

  for (const [key, ch] of Object.entries(CHANNELS)) {
    if (ch.status === "done") continue;

    if (ch.isManager) {
      const li = document.createElement("li");
      const dotClass = ch.status === "working" ? "thinking" : ch.status === "idle" ? "idle" : "";
      li.className = `agent-item manager${key === activeChannel ? " active" : ""}`;
      li.dataset.channel = key;
      li.addEventListener("click", () => setActiveChannel(key));
      li.innerHTML = `
        <div class="agent-avatar">${ch.avatar}</div>
        <div class="agent-info">
          <span class="agent-meta-line">${ch.model || ""}</span>
          <span class="agent-status-text">${ch.statusText}</span>
        </div>
        <div class="status-dot ${dotClass}"></div>
      `;
      managerList.appendChild(li);
    } else {
      orders.push({ key: key, ch: ch });
    }
  }

  // Render orders as ASCII tree
  document.getElementById("active-count").textContent = orders.length;

  for (var i = 0; i < orders.length; i++) {
    var order = orders[i];
    var isActive = order.key === activeChannel;
    var dotClass = order.ch.status === "working" ? "thinking" : order.ch.status === "idle" ? "idle" : "";
    var expanded = isActive;

    // Order heading
    var orderLine = document.createElement("div");
    orderLine.className = "tree-order" + (isActive ? " active" : "");
    orderLine.dataset.channel = order.key;
    orderLine.innerHTML =
      '<span class="tree-chevron' + (expanded ? " open" : "") + '">›</span>' +
      '<span class="tree-label">' + (order.ch.title || order.ch.name) + '</span>' +
      '<span class="status-dot ' + dotClass + '"></span>';
    treeContainer.appendChild(orderLine);

    // Stage sub-list
    var stageList = document.createElement("div");
    stageList.className = "tree-stages" + (expanded ? " open" : "");

    var stages = order.ch.stages || [];
    for (var j = 0; j < stages.length; j++) {
      var stage = stages[j];
      var stageIcon = stage.status === "done" ? "✓" : stage.status === "active" ? "●" : "○";

      var stageLine = document.createElement("div");
      stageLine.className = "tree-stage stage-" + stage.status;
      stageLine.innerHTML = '<span class="tree-icon">' + stageIcon + '</span>' + stage.name;
      stageLine.addEventListener("click", (function(k) { return function() { setActiveChannel(k); }; })(order.key));
      stageList.appendChild(stageLine);
    }

    treeContainer.appendChild(stageList);

    // Click toggles stages + selects channel
    orderLine.addEventListener("click", (function(k, sl, chevron) {
      return function() {
        setActiveChannel(k);
      };
    })(order.key, stageList, orderLine.querySelector(".tree-chevron")));
  }
}

// --- Render feed header ---

function renderFeedHeader(ch) {
  const header = document.getElementById("feed-header");
  const statusText = ch.status === "idle" ? "idle — waiting for input" : ch.status === "done" ? "completed" : "working";
  const dotClass = ch.status === "working" ? "thinking" : ch.status === "idle" ? "idle" : "active";

  let actions = "";
  if (ch.status === "idle") {
    actions = `<div class="feed-actions">
      <button class="feed-action-btn done-btn" onclick="location.href='/review.html'">DONE</button>
      <button class="feed-action-btn stop-btn" onclick="location.href='/error.html'">STOP</button>
    </div>`;
  } else if (ch.status === "working") {
    actions = `<div class="feed-actions">
      <button class="feed-action-btn stop-btn" onclick="location.href='/error.html'">STOP</button>
    </div>`;
  }

  header.innerHTML = `
    <div class="feed-title">
      ${ch.title || ch.name}
      <span class="feed-badge badge-task">${ch.taskKey}</span>
      ${ch.model ? `<span class="feed-badge">${ch.model}</span>` : ""}
      ${ch.host ? `<span class="feed-badge">${ch.host}</span>` : ""}
    </div>
    <div class="feed-meta">
      <span class="status-dot ${dotClass}"></span>
      <span>${statusText}</span>
      ${actions}
      <a href="/deploy.html" class="btn-new-order">+ ORDER</a>
    </div>
  `;
}

// --- Render messages ---

function renderMessages(ch) {
  const container = document.getElementById("feed-content");
  container.innerHTML = "";

  ch.messages.forEach((msg) => {
    if (msg.type === "system") {
      container.innerHTML += `
        <div class="message-row type-system">
          <div class="msg-avatar" style="background: transparent; border: none; font-family: var(--font-mono); color: var(--text-tertiary);">›</div>
          <div>
            <div class="msg-meta">SYSTEM</div>
            <div class="msg-body" style="font-family: var(--font-mono); font-size: 12px; color: var(--text-tertiary);">${msg.body}</div>
          </div>
        </div>`;
    } else if (msg.type === "cost") {
      container.innerHTML += `<div class="cost-line">${msg.body}</div>`;
    } else if (msg.type === "typing") {
      container.innerHTML += `
        <div class="typing-row">
          <div class="typing-indicator">thinking <span class="typing-dots"><span></span><span></span><span></span></span></div>
        </div>`;
    } else if (msg.type === "idle") {
      container.innerHTML += `
        <div class="idle-divider"><span>agent idle — waiting for input</span></div>`;
    } else if (msg.type === "tool") {
      const accentClass = {
        Read: "accent-blue",
        Grep: "accent-blue",
        Glob: "accent-blue",
        Edit: "accent-yellow",
        Write: "accent-green",
        Bash: "",
        Skill: "accent-green",
      }[msg.toolName] || "";
      container.innerHTML += `
        <div class="message-row${msg.isManager ? " from-manager" : ""}">
          <div class="msg-avatar">${msg.avatar}</div>
          <div>
            <div class="msg-meta">${msg.label} · ${msg.time} <span class="msg-badge">${msg.toolName}</span></div>
            <div class="output-block ${accentClass}">${msg.toolTarget}</div>
          </div>
        </div>`;
    } else {
      const rowClass = msg.isManager ? " from-manager" : msg.isUser ? " from-user" : "";
      const badge = msg.badge ? `<span class="msg-badge">${msg.badge}</span>` : "";
      container.innerHTML += `
        <div class="message-row${rowClass}">
          <div class="msg-avatar">${msg.avatar}</div>
          <div>
            <div class="msg-meta">${msg.label} · ${msg.time} ${badge}</div>
            <div class="msg-body">${msg.body.replace(/\n/g, "<br>")}</div>
          </div>
        </div>`;
    }
  });

  container.scrollTop = container.scrollHeight;
}

// --- Render context panel ---

function renderContext(ch) {
  const body = document.getElementById("context-body");

  // Metrics
  let html = `
    <div class="metric-grid">
      <div class="metric-card">
        <div class="metric-label">Cost</div>
        <div class="metric-value">$${ch.cost.toFixed(2)}</div>
      </div>
      <div class="metric-card">
        <div class="metric-label">Model</div>
        <div class="metric-value" style="font-size: 12px;">${ch.model || "—"}</div>
      </div>
    </div>
  `;

  // Context window
  if (ch.ctx > 0) {
    html += `
      <div class="ctx-progress">
        <div class="ctx-progress-label">
          <span>Context Window</span>
          <span>${ch.ctx}%</span>
        </div>
        <div class="ctx-progress-bar">
          <div class="ctx-progress-fill" style="width: ${ch.ctx}%;"></div>
        </div>
      </div>
    `;
  }

  // Stages
  if (ch.stages && ch.stages.length > 0) {
    html += `<div class="ctx-section-label">Pipeline</div><div class="stage-rail">`;
    ch.stages.forEach((s) => {
      const dotClass = s.status === "done" ? "done" : s.status === "active" ? "active" : "pending";
      const itemClass = s.status === "active" ? " current" : "";
      html += `<div class="stage-item${itemClass}"><span class="stage-dot ${dotClass}"></span>${s.name}</div>`;
    });
    html += "</div>";
  }

  // Files touched
  if (ch.files && ch.files.length > 0) {
    html += `<div class="ctx-section-label">Files Touched</div><div class="file-list">`;
    ch.files.forEach((f) => {
      html += `<div class="file-item"><span class="file-action ${f.action}">${f.action}</span>${f.name}</div>`;
    });
    html += "</div>";
  }

  body.innerHTML = html;
}

// --- Set active channel ---

function setActiveChannel(channel) {
  activeChannel = channel;
  const ch = CHANNELS[channel];
  if (!ch) return;

  renderSidebar();
  renderFeedHeader(ch);
  renderMessages(ch);
  renderContext(ch);

  // Update input
  const input = document.getElementById("input-field");
  if (ch.status === "done") {
    input.placeholder = "This task is complete.";
    input.disabled = true;
  } else if (ch.isManager) {
    input.placeholder = "Talk to the scheduler...";
    input.disabled = false;
  } else {
    input.placeholder = "Steer this agent...";
    input.disabled = false;
  }
}

// --- Init ---

setActiveChannel("schedule");

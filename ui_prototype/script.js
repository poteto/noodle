// ═══ noodle prototype — shared data + controls ═══

const agents = [
  { id: 'cook-alpha', name: 'Cook Alpha', type: 'execute', task: 'Fix sprites dispatcher timeout handling', model: 'opus-4-6', context: 42, duration: '4m 23s', cost: '$0.18', status: 'cooking', icon: 'flame', remote: 'sprites' },
  { id: 'cook-bravo', name: 'Cook Bravo', type: 'execute', task: 'Write tests for queue validation', model: 'opus-4-6', context: 28, duration: '2m 11s', cost: '$0.09', status: 'cooking', icon: 'flame' },
  { id: 'sous-chef', name: 'Sous Chef', type: 'prioritize', task: 'Schedule next cycle', model: 'opus-4-6', context: 61, duration: '6m 02s', cost: '$0.31', status: 'cooking', icon: 'list-ordered' },
  { id: 'taster', name: 'Taster', type: 'review', task: 'Review: fix tmux shutdown race', model: 'opus-4-6', context: 55, duration: '3m 47s', cost: '$0.22', status: 'done', icon: 'check-circle', remote: 'cursor' },
  { id: 'reflect', name: 'Reflect', type: 'reflect', task: 'Post-session reflection', model: 'opus-4-6', context: 34, duration: '2m 15s', cost: '$0.14', status: 'done', icon: 'brain' },
  { id: 'cook-charlie', name: 'Cook Charlie', type: 'execute', task: 'Watch failed.json changes', model: 'opus-4-6', context: 97, duration: '8m 41s', cost: '$0.52', status: 'failed', icon: 'x-circle', remote: 'sprites' },
];

const events = {
  'cook-alpha': [
    { t: '09:41', type: 'system', label: 'SESSION STARTED', body: 'opus-4-6 · tmux · worktree: fix-sprites-timeout' },
    { t: '09:41', type: 'think', body: 'Reading the sprites dispatcher to understand the current timeout handling. cmd.Wait() returns a non-ExitError on connection drop, causing false "failed" status.' },
    { t: '09:42', type: 'tool', label: 'Read', body: 'dispatcher/sprites_dispatcher.go' },
    { t: '09:42', type: 'tool', label: 'Read', body: 'dispatcher/sprites_session.go' },
    { t: '09:43', type: 'think', body: 'Found it. In waitAndSync, errors.As(*exec.ExitError) misses connection drops. Need to check the raw stream for a result event with subtype: "success" as fallback.' },
    { t: '09:43', type: 'tool', label: 'Grep', body: '"subtype.*success" in dispatcher/' },
    { t: '09:44', type: 'tool', label: 'Edit', body: 'dispatcher/sprites_session.go:85-102' },
    { t: '09:44', type: 'think', body: 'Added fallback: scan canonical log for a result event with "success" subtype. If found, classify as completed instead of failed.' },
    { t: '09:45', type: 'tool', label: 'Bash', body: 'go test ./dispatcher/... -run TestSpritesSessionWaitAndSync' },
    { t: '09:45', type: 'cost', body: '$0.18 · 12.4k in / 2.1k out' },
    { t: '09:46', type: 'steer', body: 'Also add a test case for when the connection drops mid-stream but the agent already pushed its branch.' },
    { t: '09:46', type: 'think', body: 'Good call. Adding a test fixture with a successful push-back event followed by a connection drop error from Wait().' },
    { t: '09:47', type: 'tool', label: 'Edit', body: 'dispatcher/sprites_session_test.go' },
  ],
  'cook-bravo': [
    { t: '09:44', type: 'system', label: 'SESSION STARTED', body: 'opus-4-6 · tmux · worktree: queue-validation-tests' },
    { t: '09:44', type: 'think', body: 'Writing tests for queue validation. Reading the existing queue package first.' },
    { t: '09:44', type: 'tool', label: 'Read', body: 'internal/queuex/queue.go' },
    { t: '09:45', type: 'tool', label: 'Read', body: 'internal/queuex/queue_test.go' },
    { t: '09:45', type: 'think', body: 'Current tests miss edge cases: duplicate IDs, empty prompts, invalid task types, concurrent writes. Adding table-driven tests.' },
    { t: '09:45', type: 'tool', label: 'Edit', body: 'internal/queuex/queue_test.go:48-52' },
    { t: '09:46', type: 'tool', label: 'Bash', body: 'go test ./internal/queuex/... -v' },
    { t: '09:46', type: 'cost', body: '$0.09 · 8.2k in / 1.4k out' },
  ],
  'sous-chef': [
    { t: '09:39', type: 'system', label: 'SESSION STARTED', body: 'opus-4-6 · tmux' },
    { t: '09:39', type: 'think', body: 'Reading mise.json — backlog, completed work, session history, failed targets.' },
    { t: '09:39', type: 'tool', label: 'Read', body: '.noodle/mise.json' },
    { t: '09:40', type: 'tool', label: 'Read', body: '.noodle/session_history.ndjson' },
    { t: '09:40', type: 'think', body: '2 recent failures on "watch failed.json" — context exhaustion. Deprioritizing. Sprites fix and queue tests are high-value.' },
    { t: '09:41', type: 'tool', label: 'Read', body: 'brain/todos.md' },
    { t: '09:42', type: 'think', body: 'Scheduling 5 items: sprites fix + queue tests (parallel execute), tmux review, reflect pass, structured logging next-up.' },
    { t: '09:42', type: 'tool', label: 'Write', body: '.noodle/queue-next.json' },
    { t: '09:43', type: 'cost', body: '$0.31 · 22.1k in / 3.8k out' },
  ],
  'taster': [
    { t: '09:28', type: 'system', label: 'SESSION STARTED', body: 'opus-4-6 · tmux' },
    { t: '09:28', type: 'think', body: 'Reviewing tmux shutdown fix. Checking diff, verifying tests, assessing root cause.' },
    { t: '09:28', type: 'tool', label: 'Bash', body: 'git diff main...fix-tmux-shutdown' },
    { t: '09:29', type: 'think', body: 'Fix adds a mutex around the summary buffer write. Race was between collectOutput goroutine and Wait() returning. Clean.' },
    { t: '09:30', type: 'tool', label: 'Bash', body: 'go test ./dispatcher/... -race' },
    { t: '09:31', type: 'think', body: 'All tests pass including race detector. Minimal and correct. Approving.' },
    { t: '09:31', type: 'tool', label: 'Write', body: '.noodle/quality/execute-fix-tmux-shutdown.json' },
    { t: '09:32', type: 'system', label: 'APPROVED', body: 'merged to main' },
    { t: '09:32', type: 'cost', body: '$0.22 · 15.6k in / 2.9k out' },
  ],
  'reflect': [
    { t: '09:33', type: 'system', label: 'SESSION STARTED', body: 'opus-4-6 · tmux' },
    { t: '09:33', type: 'think', body: 'Reflecting on tmux shutdown race fix. Missing mutex — common Go dispatcher pattern. Worth documenting.' },
    { t: '09:34', type: 'tool', label: 'Read', body: 'brain/codebase/dispatchers.md' },
    { t: '09:34', type: 'think', body: 'Shared buffers between collection goroutine and Wait path need synchronization. Adding note.' },
    { t: '09:35', type: 'tool', label: 'Edit', body: 'brain/codebase/tmux-shutdown.md' },
    { t: '09:35', type: 'system', label: 'COMPLETE', body: 'updated 1 brain file' },
    { t: '09:35', type: 'cost', body: '$0.14 · 9.8k in / 1.7k out' },
  ],
  'cook-charlie': [
    { t: '09:15', type: 'system', label: 'SESSION STARTED', body: 'opus-4-6 · tmux · worktree: watch-failed-json' },
    { t: '09:15', type: 'think', body: 'Implementing fsnotify watching for failed.json. Exploring current failed target handling.' },
    { t: '09:15', type: 'tool', label: 'Read', body: 'loop/loop.go' },
    { t: '09:16', type: 'tool', label: 'Read', body: 'loop/failed.go' },
    { t: '09:16', type: 'tool', label: 'Grep', body: '"fsnotify" in go.mod' },
    { t: '09:17', type: 'think', body: 'fsnotify already a dependency. Adding watcher in Run method.' },
    { t: '09:18', type: 'tool', label: 'Edit', body: 'loop/loop.go:45-67' },
    { t: '09:19', type: 'tool', label: 'Edit', body: 'loop/failed.go:12-30' },
    { t: '09:20', type: 'tool', label: 'Bash', body: 'go test ./loop/... -v' },
    { t: '09:20', type: 'think', body: 'Tests failing — watcher fires too eagerly during write. Need debounce...' },
    { t: '09:21', type: 'tool', label: 'Edit', body: 'loop/loop.go:48-72' },
    { t: '09:22', type: 'tool', label: 'Edit', body: 'loop/loop_test.go:88-105' },
    { t: '09:23', type: 'cost', body: '$0.52 · 41.2k in / 6.3k out' },
    { t: '09:24', type: 'system', label: 'FAILED', body: 'context window exhausted at 97%' },
  ],
};

const queue = [
  { n: 1, type: 'execute', title: 'Fix sprites timeout', status: 'cooking' },
  { n: 2, type: 'execute', title: 'Queue validation tests', status: 'cooking' },
  { n: 3, type: 'prioritize', title: 'Schedule next cycle', status: 'cooking' },
  { n: 4, type: 'execute', title: 'Structured loop logging', status: 'planned' },
  { n: 5, type: 'execute', title: 'PID file stale detection', status: 'planned' },
  { n: 6, type: 'review', title: 'Review: tmux shutdown fix', status: 'ready' },
  { n: 7, type: 'reflect', title: 'Post-session reflection', status: 'ready' },
  { n: 8, type: 'execute', title: 'Watch failed.json changes', status: 'blocked' },
];

// ── Theme Switching ──────────────────────────────

function initThemes() {
  setTheme('poster');
}

function setTheme(name) {
  document.documentElement.setAttribute('data-theme', name);
  localStorage.setItem('noodle-theme', name);
  document.querySelectorAll('[data-theme-btn]').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.themeBtn === name);
  });
}

// ── Nav Highlighting ─────────────────────────────

function initNav() {
  const page = location.pathname.split('/').pop() || 'index.html';
  document.querySelectorAll('.proto-nav a').forEach(a => {
    const href = a.getAttribute('href');
    if (href === page || (page === 'index.html' && href === 'index.html')) {
      a.classList.add('active');
    }
  });
}

// ── Shared Init ──────────────────────────────────

document.addEventListener('DOMContentLoaded', () => {
  initThemes();
  initNav();
  if (window.lucide) lucide.createIcons();
});

// ── Helpers ──────────────────────────────────────

function esc(s) {
  const d = document.createElement('div');
  d.textContent = s;
  return d.innerHTML;
}

function typeColor(type) {
  const map = { execute: 'green', plan: 'blue', review: 'yellow', reflect: 'pink', prioritize: 'orange' };
  return map[type] || 'text-2';
}

function statusColor(status) {
  const map = { cooking: 'green', done: 'green', failed: 'red', planned: 'text-2', ready: 'blue', blocked: 'orange' };
  return map[status] || 'text-3';
}

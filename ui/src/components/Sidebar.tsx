export function Sidebar() {
  return (
    <aside className="flex flex-col border-r border-border-subtle bg-bg-surface h-full overflow-hidden">
      <div className="p-4 border-b border-border-subtle">
        <h1 className="text-lg font-display font-bold tracking-wider uppercase">NOODLE</h1>
      </div>
      <nav className="p-2 border-b border-border-subtle">
        {/* Nav links placeholder */}
      </nav>
      <div className="flex-1 overflow-y-auto p-2">
        {/* Channel list placeholder */}
      </div>
      <div className="p-3 border-t border-border-subtle text-xs text-neutral-500">
        {/* Footer placeholder */}
      </div>
    </aside>
  );
}

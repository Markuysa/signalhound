import type { ReactNode } from 'react';

/* The 380px detail panel. Hidden below 1180px (DESIGN.md §2.3); the screen tickets fill
   it. Kept here so the shell owns the breakpoint, not each page. */
export function DetailPanel({ children }: { children?: ReactNode }) {
  return (
    <aside className="hidden w-detail shrink-0 border-l border-border-soft bg-surface min-[1180px]:block">
      {children}
    </aside>
  );
}

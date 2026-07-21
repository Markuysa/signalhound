import type { ReactNode } from 'react';
import { Inbox } from 'lucide-react';

interface Props {
  title: string;
  hint?: string;
  icon?: ReactNode;
}

/* Empty states carry a hint, per PRD §3.3 — they matter for first-run impression. */
export function EmptyState({ title, hint, icon }: Props) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 py-16 text-center">
      <div className="text-text-dim">{icon ?? <Inbox size={28} strokeWidth={1.5} />}</div>
      <p className="text-[14px] text-text-mut">{title}</p>
      {hint && <p className="max-w-xs text-[13px] text-text-dim">{hint}</p>}
    </div>
  );
}

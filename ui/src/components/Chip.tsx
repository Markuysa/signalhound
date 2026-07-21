import type { ReactNode } from 'react';

interface Props {
  children: ReactNode;
  active?: boolean;
  onClick?: () => void;
}

/* Section labels and filters. Mono uppercase per DESIGN.md §2.2. */
export function Chip({ children, active = false, onClick }: Props) {
  const Tag = onClick ? 'button' : 'span';
  return (
    <Tag
      onClick={onClick}
      className={[
        'inline-flex items-center rounded-chip px-2.5 py-1 font-mono text-[11px] uppercase tracking-wide transition-colors duration-150',
        active ? 'bg-amber-soft text-amber' : 'text-text-dim hover:text-text-mut',
      ].join(' ')}
    >
      {children}
    </Tag>
  );
}

import type { ButtonHTMLAttributes, ReactNode } from 'react';

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'ghost';
  children: ReactNode;
}

export function Button({ variant = 'ghost', children, className = '', ...rest }: Props) {
  const base =
    'inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-[13px] font-medium transition-colors duration-150 disabled:opacity-50';
  const styles =
    variant === 'primary'
      ? 'bg-amber text-bg hover:bg-amber/90'
      : 'border border-border text-text-mut hover:bg-surface-2 hover:text-text';
  return (
    <button className={`${base} ${styles} ${className}`} {...rest}>
      {children}
    </button>
  );
}

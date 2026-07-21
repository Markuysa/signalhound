import type { Config } from 'tailwindcss';

/* Tokens are defined once in src/styles/tokens.css as CSS variables; Tailwind only
   references them, so there is exactly one place a colour is ever written (DESIGN.md §2). */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    colors: {
      transparent: 'transparent',
      bg: 'var(--bg)',
      surface: 'var(--surface)',
      'surface-2': 'var(--surface-2)',
      border: 'var(--border)',
      'border-soft': 'var(--border-soft)',
      text: 'var(--text)',
      'text-mut': 'var(--text-mut)',
      'text-dim': 'var(--text-dim)',
      amber: 'var(--amber)',
      'amber-soft': 'var(--amber-soft)',
      hot: 'var(--hot)',
      live: 'var(--live)',
      danger: 'var(--danger)',
    },
    borderRadius: { chip: '999px', DEFAULT: 'var(--r)', sm: '6px', md: '8px' },
    fontFamily: {
      display: ['Space Grotesk', 'sans-serif'],
      sans: ['Inter', 'system-ui', 'sans-serif'],
      mono: ['JetBrains Mono', 'monospace'],
    },
    extend: {
      spacing: { sidebar: 'var(--sidebar-w)', detail: 'var(--detail-w)' },
      keyframes: {
        'live-pulse': {
          '0%,100%': { boxShadow: '0 0 0 0 var(--amber-soft)' },
          '50%': { boxShadow: '0 0 0 4px transparent' },
        },
      },
      animation: { 'live-pulse': 'live-pulse 2s ease-in-out infinite' },
    },
  },
  plugins: [],
} satisfies Config;

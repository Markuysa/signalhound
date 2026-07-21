import { NavLink } from 'react-router-dom';
import { LayoutDashboard, Radio, Settings } from 'lucide-react';
import { BudgetBar } from './BudgetBar';

/* Fixed 216px sidebar (DESIGN.md §2.3), dark only. Collapses to a top menu below 820px
   via the responsive utilities; the LLM budget bar lives in the footer. */
const NAV = [
  { to: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/feed', label: 'Feed', icon: Radio },
  { to: '/settings', label: 'Settings', icon: Settings },
];

export function Sidebar() {
  return (
    <aside className="flex w-full shrink-0 flex-col border-b border-border-soft bg-surface min-[820px]:h-full min-[820px]:w-sidebar min-[820px]:border-b-0 min-[820px]:border-r">
      <div className="flex items-center gap-2 px-4 py-4">
        <Logo />
        <span className="font-display text-[17px] text-text">SignalHound</span>
      </div>

      <nav className="flex flex-row gap-1 px-2 min-[820px]:flex-col">
        {NAV.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              [
                'flex items-center gap-2.5 rounded-md px-3 py-2 text-[13px] transition-colors duration-150',
                isActive
                  ? 'bg-amber-soft text-amber'
                  : 'text-text-mut hover:bg-surface-2 hover:text-text',
              ].join(' ')
            }
          >
            <Icon size={16} strokeWidth={1.75} />
            <span className="hidden min-[820px]:inline">{label}</span>
          </NavLink>
        ))}
      </nav>

      <div className="mt-auto hidden border-t border-border-soft p-3 min-[820px]:block">
        <BudgetBar />
      </div>
    </aside>
  );
}

/* The logo is the signal broken-line with a hot dot — the meter motif at small scale. */
function Logo() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" aria-hidden>
      <path d="M2 14 L7 9 L11 12 L18 4" stroke="var(--amber)" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" />
      <circle cx="18" cy="4" r="2.2" fill="var(--hot)" />
    </svg>
  );
}

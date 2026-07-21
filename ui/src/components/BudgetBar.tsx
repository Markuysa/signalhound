interface Props {
  spent?: number;
  budget?: number;
}

/* The daily LLM budget, shown in the sidebar footer. Turns hot as it approaches the cap
   so an over-budget run is visible at a glance. Values arrive from GET /api/stats once
   the Dashboard ticket wires them; here it renders a resting state. */
export function BudgetBar({ spent = 0, budget = 0 }: Props) {
  const pct = budget > 0 ? Math.min(100, (spent / budget) * 100) : 0;
  const near = pct >= 80;
  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center justify-between font-mono text-[10.5px] uppercase tracking-wide text-text-dim">
        <span>LLM budget</span>
        <span className="text-text-mut">
          ${spent.toFixed(2)} / ${budget.toFixed(2)}
        </span>
      </div>
      <div className="h-1 overflow-hidden rounded-chip bg-surface-2">
        <div
          className={near ? 'h-full bg-hot' : 'h-full bg-amber'}
          style={{ width: `${pct}%`, transition: 'width 150ms' }}
        />
      </div>
    </div>
  );
}

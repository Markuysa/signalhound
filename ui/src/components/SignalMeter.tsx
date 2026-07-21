/* The signal meter: a lead's score as five ascending strokes plus a mono numeral, the
   brand's signature element (DESIGN.md §3.1). Fill and colour bands are exact:

     0–40  → 2 strokes, text-dim
     41–60 → 3 strokes, amber
     61–84 → 4 strokes, amber
     85+   → 5 strokes, hot

   Unfilled strokes stay visible in --border so it reads as a scale, not bars. */

export type SignalMeterSize = 'standard' | 'small';

const STROKE_COUNT = 5;

interface Band {
  filled: number;
  color: 'text-dim' | 'amber' | 'hot';
}

export function bandForScore(score: number): Band {
  if (score >= 85) return { filled: 5, color: 'hot' };
  if (score >= 61) return { filled: 4, color: 'amber' };
  if (score >= 41) return { filled: 3, color: 'amber' };
  return { filled: 2, color: 'text-dim' };
}

const SIZES: Record<SignalMeterSize, { height: number; stroke: number; font: number }> = {
  standard: { height: 22, stroke: 5, font: 13 },
  small: { height: 14, stroke: 3.5, font: 11 },
};

const FILL_VAR: Record<Band['color'], string> = {
  'text-dim': 'var(--text-dim)',
  amber: 'var(--amber)',
  hot: 'var(--hot)',
};

interface Props {
  score: number;
  size?: SignalMeterSize;
  showValue?: boolean;
}

export function SignalMeter({ score, size = 'standard', showValue = true }: Props) {
  const { height, stroke, font } = SIZES[size];
  const { filled, color } = bandForScore(score);
  const gap = stroke * 0.7;

  return (
    <div
      className="inline-flex flex-col items-center gap-0.5"
      role="img"
      aria-label={`Signal score ${score} of 100`}
      data-testid="signal-meter"
      data-score={score}
      data-filled={filled}
      data-color={color}
    >
      <div className="flex items-end" style={{ height, gap }}>
        {Array.from({ length: STROKE_COUNT }, (_, i) => {
          const isFilled = i < filled;
          // Ascending: each stroke is taller than the last, like reception bars.
          const h = height * (0.4 + (0.6 * (i + 1)) / STROKE_COUNT);
          return (
            <span
              key={i}
              style={{
                width: stroke,
                height: h,
                borderRadius: 2,
                background: isFilled ? FILL_VAR[color] : 'var(--border)',
              }}
            />
          );
        })}
      </div>
      {showValue && (
        <span className="font-mono tabular-nums text-text-mut" style={{ fontSize: font }}>
          {score}
        </span>
      )}
    </div>
  );
}

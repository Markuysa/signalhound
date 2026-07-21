import { render } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { SignalMeter, bandForScore } from './SignalMeter';

/* The fill bands are the brand's core visual promise (DESIGN.md §3.1), so they are
   pinned at the exact scores the ticket names: 12, 45, 70, 92. */
describe('SignalMeter bands', () => {
  it.each([
    [12, 2, 'text-dim'],
    [45, 3, 'amber'],
    [70, 4, 'amber'],
    [92, 5, 'hot'],
  ])('score %i → %i strokes, %s', (score, filled, color) => {
    expect(bandForScore(score)).toEqual({ filled, color });
  });

  it('band boundaries are exact', () => {
    expect(bandForScore(40).filled).toBe(2);
    expect(bandForScore(41).filled).toBe(3);
    expect(bandForScore(60).filled).toBe(3);
    expect(bandForScore(61).filled).toBe(4);
    expect(bandForScore(84).filled).toBe(4);
    expect(bandForScore(85)).toEqual({ filled: 5, color: 'hot' });
  });

  it.each([12, 45, 70, 92])('renders a stable snapshot at %i', (score) => {
    const { container } = render(<SignalMeter score={score} />);
    expect(container.firstChild).toMatchSnapshot();
  });

  it('exposes score and band via data attributes and an accessible label', () => {
    const { getByTestId } = render(<SignalMeter score={92} size="small" />);
    const el = getByTestId('signal-meter');
    expect(el).toHaveAttribute('data-filled', '5');
    expect(el).toHaveAttribute('data-color', 'hot');
    expect(el).toHaveAttribute('aria-label', 'Signal score 92 of 100');
  });
});

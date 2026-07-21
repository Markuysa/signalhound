import { useEffect, useRef, useState } from 'react';

/* Live event stream from GET /api/events. The Go side emits signal.scored,
   budget.exceeded and collector.error (ARCHITECTURE.md contract). */
export type SSEEventType = 'signal.scored' | 'budget.exceeded' | 'collector.error';

export interface SSEEvent<T = unknown> {
  type: SSEEventType;
  data: T;
}

export function useEventStream(onEvent: (e: SSEEvent) => void, enabled = true) {
  const [connected, setConnected] = useState(false);
  const handler = useRef(onEvent);
  handler.current = onEvent;

  useEffect(() => {
    if (!enabled) return;
    const es = new EventSource('/api/events', { withCredentials: true });
    es.onopen = () => setConnected(true);
    es.onerror = () => setConnected(false);

    const types: SSEEventType[] = ['signal.scored', 'budget.exceeded', 'collector.error'];
    for (const t of types) {
      es.addEventListener(t, (ev) => {
        const data = (ev as MessageEvent).data;
        handler.current({ type: t, data: safeParse(data) });
      });
    }
    return () => es.close();
  }, [enabled]);

  return { connected };
}

function safeParse(s: string): unknown {
  try {
    return JSON.parse(s);
  } catch {
    return s;
  }
}

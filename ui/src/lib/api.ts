/* Typed client for the API contract frozen in docs/ARCHITECTURE.md. The UI codes against
   this before the backend exists; the Vite dev server proxies /api to the Go process.
   Auth is the session cookie set by POST /api/session, so requests send credentials and
   carry no token themselves. */

export type IntentType = 'buying' | 'pain' | 'hiring' | 'research' | 'none';
export type LeadStatus = 'new' | 'reviewed' | 'contacted' | 'ignored' | 'customer';

export interface Signal {
  id: string;
  source: string;
  url: string;
  author: string;
  title: string;
  content: string;
  lang: string;
  published_at: string;
  collected_at: string;
  content_hash: string;
}

export interface Score {
  signal_id: string;
  value: number;
  intent_type: IntentType;
  reasons: string[];
  icp_match: string[];
  draft_reply: string;
  model: string;
  cost_usd: number;
  latency_ms: number;
  scored_at: string;
}

export interface SignalWithScore {
  signal: Signal;
  score: Score | null;
}

export interface Lead {
  id: string;
  name: string;
  company: string;
  signals: string[];
  best_score: number;
  status: LeadStatus;
  notes: string;
}

export interface Stats {
  signals_per_day: { day: string; count: number }[];
  spend_usd: number;
  score_histogram: { bucket: string; count: number }[];
  source_funnel: { source: string; signals: number; hot: number }[];
}

export interface SignalsQuery {
  score_gte?: number;
  source?: string;
  intent?: IntentType;
  period?: string;
  cursor?: string;
  limit?: number;
}

export interface Page<T> {
  items: T[];
  next_cursor: string;
}

export interface ScoreTestResult {
  score: number;
  intent_type: IntentType;
  reasons: string[];
  icp_match: string[];
  cost_usd: number;
  latency_ms: number;
}

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
    readonly fields?: Record<string, string>,
  ) {
    super(message);
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    credentials: 'same-origin', // the session cookie from POST /api/session
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!res.ok) {
    let fields: Record<string, string> | undefined;
    let message = res.statusText;
    try {
      const data = await res.json();
      message = data.error ?? message;
      fields = data.fields;
    } catch {
      /* non-JSON error body */
    }
    throw new ApiError(res.status, message, fields);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

function query(params: Record<string, unknown>): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '' && v !== null) q.set(k, String(v));
  }
  const s = q.toString();
  return s ? `?${s}` : '';
}

export const api = {
  createSession: () => request<void>('POST', '/api/session'),

  signals: (q: SignalsQuery = {}) =>
    request<Page<Signal>>('GET', `/api/signals${query({ ...q })}`),
  signal: (id: string) => request<SignalWithScore>('GET', `/api/signals/${id}`),

  leads: () => request<Lead[]>('GET', '/api/leads'),
  updateLead: (id: string, patch: { status?: LeadStatus; notes?: string }) =>
    request<Lead>('PATCH', `/api/leads/${id}`, patch),

  stats: () => request<Stats>('GET', '/api/stats'),

  config: () => request<{ yaml: string; parsed: unknown }>('GET', '/api/config'),
  saveConfig: (yaml: string) => request<void>('PUT', '/api/config', { yaml }),

  testScore: (text: string) => request<ScoreTestResult>('POST', '/api/score/test', { text }),
};

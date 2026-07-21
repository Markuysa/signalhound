import { describe, it, expect, vi, beforeEach } from 'vitest';
import { api, ApiError } from './api';

/* The client is the frozen contract the screen tickets build on, so its request shapes
   are pinned: correct method, path, query encoding, and the session cookie. */
describe('api client', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  function mockFetch(status: number, body: unknown) {
    return vi.fn().mockResolvedValue({
      ok: status >= 200 && status < 300,
      status,
      statusText: 'x',
      json: async () => body,
    });
  }

  it('sends credentials so the session cookie rides along', async () => {
    const f = mockFetch(200, { items: [], next_cursor: '' });
    vi.stubGlobal('fetch', f);
    await api.signals();
    expect(f).toHaveBeenCalledWith('/api/signals', expect.objectContaining({ credentials: 'same-origin' }));
  });

  it('encodes only the filters that are set', async () => {
    const f = mockFetch(200, { items: [], next_cursor: '' });
    vi.stubGlobal('fetch', f);
    await api.signals({ score_gte: 70, source: 'hackernews', limit: 25 });
    const url = f.mock.calls[0][0] as string;
    expect(url).toBe('/api/signals?score_gte=70&source=hackernews&limit=25');
  });

  it('covers every contract endpoint', () => {
    for (const m of ['createSession', 'signals', 'signal', 'leads', 'updateLead', 'stats', 'config', 'saveConfig', 'testScore'] as const) {
      expect(typeof api[m]).toBe('function');
    }
  });

  it('raises ApiError with field errors on 422', async () => {
    vi.stubGlobal('fetch', mockFetch(422, { error: 'invalid', fields: { 'llm.model': 'required' } }));
    await expect(api.saveConfig('bad')).rejects.toMatchObject({
      status: 422,
      fields: { 'llm.model': 'required' },
    });
    await expect(api.saveConfig('bad')).rejects.toBeInstanceOf(ApiError);
  });
});

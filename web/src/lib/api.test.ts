// SPDX-License-Identifier: GPL-3.0-or-later

import { afterEach, describe, expect, it, vi } from 'vitest';
import { api, ApiError } from './api';

function mockFetch(status: number, body: unknown) {
  const fn = vi.fn().mockResolvedValue(
    new Response(body === undefined ? null : JSON.stringify(body), {
      status,
      headers: { 'Content-Type': 'application/json' },
    }),
  );
  vi.stubGlobal('fetch', fn);
  return fn;
}

afterEach(() => vi.unstubAllGlobals());

describe('api', () => {
  it('returns parsed JSON', async () => {
    mockFetch(200, { email: 'a@b.c' });
    expect(await api('GET', '/api/session')).toEqual({ email: 'a@b.c' });
  });

  it('sends JSON bodies', async () => {
    const fn = mockFetch(200, {});
    await api('POST', '/api/login', { email: 'a' });
    const [, init] = fn.mock.calls[0];
    expect(init.method).toBe('POST');
    expect(init.headers['Content-Type']).toBe('application/json');
    expect(init.body).toBe('{"email":"a"}');
    expect(init.credentials).toBe('same-origin');
  });

  it('handles 204', async () => {
    mockFetch(204, undefined);
    expect(await api('POST', '/api/logout')).toBeUndefined();
  });

  it('throws the server error code', async () => {
    mockFetch(401, { error: 'auth.invalid_credentials' });
    await expect(api('POST', '/api/login', {})).rejects.toMatchObject({ code: 'auth.invalid_credentials', status: 401 });
  });

  it('falls back to a generic code', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('<html>', { status: 502 })));
    await expect(api('GET', '/api/x')).rejects.toMatchObject({ code: 'server.error' });
  });

  it('maps network failures', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('offline')));
    const err = (await api('GET', '/api/x').catch((e: unknown) => e)) as ApiError;
    expect(err).toBeInstanceOf(ApiError);
    expect(err.code).toBe('network');
  });
});

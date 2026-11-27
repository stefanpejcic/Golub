// SPDX-License-Identifier: GPL-3.0-or-later

import { expect, test, type APIRequestContext } from '@playwright/test';
import { ALICE, PASSWORD } from './env';

// Budgets are server-side times from Server-Timing, so the network does not count against them.
// Change them only on purpose, never to make a slow build pass.
const LIST_MS = Number(process.env.PERF_LIST_MS ?? 300);
const OPEN_MS = Number(process.env.PERF_OPEN_MS ?? 200);
const RUNS = 5;

async function serverMs(api: APIRequestContext, path: string): Promise<number> {
  const res = await api.get(path);
  expect(res.status(), path).toBe(200);
  const m = /app;dur=([\d.]+)/.exec(res.headers()['server-timing'] ?? '');
  expect(m, `Server-Timing on ${path}`).not.toBeNull();
  return Number(m![1]);
}

async function median(api: APIRequestContext, path: string): Promise<number> {
  await serverMs(api, path);
  const times: number[] = [];
  for (let i = 0; i < RUNS; i++) times.push(await serverMs(api, path));
  times.sort((a, b) => a - b);
  return times[Math.floor(RUNS / 2)];
}

test.describe('performance budgets on a 50k folder', () => {
  test.skip(!!process.env.SKIP_PERF, 'performance checks disabled');

  test('list and open stay within budget', async ({ playwright, baseURL }) => {
    const api = await playwright.request.newContext({ baseURL });
    const login = await api.post('/api/login', { data: { email: ALICE, password: PASSWORD } });
    expect(login.status()).toBe(200);

    const first = await api.get('/api/messages?folder=Perf&page=1&size=50');
    const body = await first.json();
    test.skip(body.total < 50000, `Perf folder has ${body.total} messages; run perf-seed.sh`);

    const page1 = await median(api, '/api/messages?folder=Perf&page=1&size=50');
    const deep = await median(api, '/api/messages?folder=Perf&page=500&size=50');
    const uid = body.messages[0].uid;
    const open = await median(api, `/api/message?folder=Perf&uid=${uid}&seen=0`);
    const search = await median(api, '/api/messages?folder=Perf&page=1&size=50&q=Perf%20message%20049');
    console.log(`perf: first page ${page1}ms, page 500 ${deep}ms, open ${open}ms, search ${search}ms`);

    expect(page1).toBeLessThan(LIST_MS);
    expect(deep).toBeLessThan(LIST_MS);
    expect(open).toBeLessThan(OPEN_MS);
    await api.dispose();
  });
});

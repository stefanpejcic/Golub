// SPDX-License-Identifier: GPL-3.0-or-later
// Logs bench1..benchN into a webmail client in a real browser and records the message-list request its UI sends,
// so load.sh can replay exactly that request for each account.
// Usage: E2E_PASSWORD=... node bench/sessions.mjs <snappymail|afterlogic> <url> [accounts] > sessions-<client>.json
import { chromium } from '@playwright/test';

const [client, base, countArg] = process.argv.slice(2);
const N = Number(countArg ?? 10);
const password = process.env.E2E_PASSWORD;

const clients = {
  snappymail: {
    login: async (p, user) => {
      await p.locator('input[name=Email]').fill(user);
      await p.locator('input[name=Password]').fill(password);
      await p.locator('.buttonLogin').click();
    },
    // the first login asks for an identity; saving it once keeps the popup away from the benchmark runs
    afterLogin: async (p) => {
      const ident = p.locator('dialog:has-text("Edit Identity")');
      if (await ident.isVisible().catch(() => false)) await ident.locator('button:has-text("Save")').click();
    },
    openPerf: (p) => p.locator('.b-folders-user').getByText('Perf', { exact: true }).click(),
    isList: (r) => r.method() === 'POST' && (r.postData() ?? '').includes('"Action":"MessageList"'),
    // same page of 50 for every client, without threads
    body: (raw, folder) => JSON.stringify({ ...JSON.parse(raw), folder, limit: 50, offset: 0, threadUid: 0 }),
  },
  afterlogic: {
    login: async (p, user) => {
      await p.locator('#login_login').fill(user);
      await p.locator('#login_password').fill(password);
      await p.getByText('Sign In').click();
    },
    afterLogin: async () => {},
    openPerf: (p) => p.getByText('Perf', { exact: true }).first().click(),
    isList: (r) => r.method() === 'POST' && (r.postData() ?? '').includes('Method=GetMessages&'),
    body: (raw, folder) => {
      const form = new URLSearchParams(raw);
      const params = { ...JSON.parse(form.get('Parameters')), Folder: folder, Offset: 0, Limit: 50, UseThreading: false };
      form.set('Parameters', JSON.stringify(params));
      return form.toString();
    },
  },
};

const c = clients[client];
if (!c || !base || !password) throw new Error('usage: E2E_PASSWORD=... node bench/sessions.mjs <snappymail|afterlogic> <url> [accounts]');
const browser = await chromium.launch();
const out = [];
for (let i = 1; i <= N; i++) {
  const ctx = await browser.newContext({ viewport: { width: 1400, height: 900 }, locale: 'en-US' });
  const page = await ctx.newPage();
  await page.goto(base);
  await c.login(page, `bench${i}@golub.test`);
  await page.waitForTimeout(5000);
  await c.afterLogin(page);
  const listed = page.waitForRequest(c.isList, { timeout: 30_000 });
  await c.openPerf(page);
  const req = await listed;
  const headers = Object.fromEntries(Object.entries(req.headers()).filter(([k]) => /^(x-|content-type$|authorization$)/i.test(k)));
  const cookie = (await ctx.cookies()).map((k) => `${k.name}=${k.value}`).join('; ');
  out.push({
    url: new URL(req.url()).pathname + new URL(req.url()).search,
    headers,
    cookie,
    bodies: { INBOX: c.body(req.postData(), 'INBOX'), Perf: c.body(req.postData(), 'Perf') },
  });
  await ctx.close();
}
await browser.close();
console.log(JSON.stringify(out, null, 2));

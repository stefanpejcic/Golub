// SPDX-License-Identifier: GPL-3.0-or-later
// Compares Golub and Roundcube against the same mailbox: page weight, requests and user-visible timings.
// Usage: node bench/bench.mjs <golub-url> <roundcube-url> [runs]
import { chromium } from '@playwright/test';

const [golubURL, rcURL, runsArg] = process.argv.slice(2);
const RUNS = Number(runsArg ?? 3);
const USER = 'alice@golub.test';
const PASS = 'golub-e2e-password';

const clients = {
  golub: {
    url: golubURL,
    login: async (p) => {
      await p.getByLabel('Email address').fill(USER);
      await p.getByLabel('Password').fill(PASS);
      await p.getByRole('button', { name: 'Sign in' }).click();
    },
    inbox: (p) => p.getByTestId('message').first(),
    openFolder: (p) => p.locator('[data-testid=folder][data-folder=Perf]').click(),
    folderRow: (p) => p.getByTestId('message').filter({ hasText: 'Perf message' }).first(),
    openMessage: (p) => p.getByTestId('message').filter({ hasText: 'Perf message' }).first().getByRole('link').click(),
    messageShown: (p) => p.getByTestId('message-text'),
  },
  roundcube: {
    url: rcURL,
    login: async (p) => {
      await p.locator('#rcmloginuser').fill(USER);
      await p.locator('#rcmloginpwd').fill(PASS);
      await p.locator('#rcmloginsubmit').click();
    },
    inbox: (p) => p.locator('#messagelist tbody tr').first(),
    openFolder: (p) => p.locator('#mailboxlist a[rel="Perf"]').click(),
    folderRow: (p) => p.locator('#messagelist tbody tr', { hasText: 'Perf message' }).first(),
    openMessage: (p) => p.locator('#messagelist tbody tr', { hasText: 'Perf message' }).first().click(),
    messageShown: (p) => p.frameLocator('#messagecontframe').locator('#messagebody'),
  },
};

const median = (xs) => [...xs].sort((a, b) => a - b)[Math.floor(xs.length / 2)];

async function measure(browser, c) {
  const ctx = await browser.newContext({ locale: 'en-US', viewport: { width: 1400, height: 900 } });
  const page = await ctx.newPage();
  let requests = 0;
  let bytes = 0;
  page.on('request', () => requests++);
  page.on('requestfinished', async (r) => {
    const s = await r.sizes().catch(() => null);
    if (s) bytes += s.responseBodySize + s.responseHeadersSize;
  });

  let t = Date.now();
  await page.goto(c.url, { waitUntil: 'load' });
  const loginPageMs = Date.now() - t;
  await page.waitForTimeout(300);
  const loginPage = { bytes, requests };

  t = Date.now();
  await c.login(page);
  await c.inbox(page).waitFor({ timeout: 30000 });
  const loginToInboxMs = Date.now() - t;

  t = Date.now();
  await c.openFolder(page);
  await c.folderRow(page).waitFor({ timeout: 60000 });
  const openFolderMs = Date.now() - t;

  t = Date.now();
  await c.openMessage(page);
  await c.messageShown(page).waitFor({ timeout: 30000 });
  const openMessageMs = Date.now() - t;

  await page.waitForTimeout(500);
  const result = { loginPageMs, loginPage, loginToInboxMs, openFolderMs, openMessageMs, sessionBytes: bytes, sessionRequests: requests };
  await ctx.close();
  return result;
}

const browser = await chromium.launch();
const out = {};
for (const [name, c] of Object.entries(clients)) {
  const runs = [];
  for (let i = 0; i < RUNS; i++) runs.push(await measure(browser, c));
  out[name] = {
    loginPageKB: Math.round(median(runs.map((r) => r.loginPage.bytes)) / 1024),
    loginPageRequests: median(runs.map((r) => r.loginPage.requests)),
    loginToInboxMs: median(runs.map((r) => r.loginToInboxMs)),
    openFolder50kMs: median(runs.map((r) => r.openFolderMs)),
    openMessageMs: median(runs.map((r) => r.openMessageMs)),
    sessionKB: Math.round(median(runs.map((r) => r.sessionBytes)) / 1024),
    sessionRequests: median(runs.map((r) => r.sessionRequests)),
  };
}
await browser.close();
console.log(JSON.stringify(out, null, 2));

// SPDX-License-Identifier: GPL-3.0-or-later
// Compares webmail clients against the same mailbox in Chromium: page weight, requests and user-visible timings.
// Usage: E2E_PASSWORD=... node bench/bench.mjs golub=<url> roundcube=<url> snappymail=<url> afterlogic=<url> [runs=5]
import { chromium } from '@playwright/test';

const args = Object.fromEntries(process.argv.slice(2).map((a) => a.split('=')));
const RUNS = Number(args.runs ?? 5);
const USER = args.user ?? 'bench1@golub.test';
const PASS = process.env.E2E_PASSWORD;

const clients = {
  golub: {
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
  snappymail: {
    login: async (p) => {
      await p.locator('input[name=Email]').fill(USER);
      await p.locator('input[name=Password]').fill(PASS);
      await p.locator('.buttonLogin').click();
    },
    inbox: (p) => p.locator('.messageListItem').first(),
    // a first login on a new account asks for an identity once
    beforeFolder: async (p) => {
      const ident = p.locator('dialog:has-text("Edit Identity")');
      if (await ident.isVisible().catch(() => false)) await ident.locator('button:has-text("Save")').click();
    },
    openFolder: (p) => p.locator('.b-folders-user').getByText('Perf', { exact: true }).click(),
    folderRow: (p) => p.locator('.messageListItem', { hasText: 'Perf message' }).first(),
    openMessage: (p) => p.locator('.messageListItem', { hasText: 'Perf message' }).first().click(),
    messageShown: (p) => p.locator('#messageItem .bodyText'),
  },
  afterlogic: {
    login: async (p) => {
      await p.locator('#login_login').fill(USER);
      await p.locator('#login_password').fill(PASS);
      await p.getByText('Sign In').click();
    },
    inbox: (p) => p.getByText('Plain text', { exact: true }).first(),
    openFolder: (p) => p.getByText('Perf', { exact: true }).first().click(),
    folderRow: (p) => p.getByText(/^Perf message/).first(),
    openMessage: (p) => p.getByText(/^Perf message/).first().click(),
    messageShown: (p) => p.locator('.message_body'),
  },
};

const median = (xs) => [...xs].sort((a, b) => a - b)[Math.floor(xs.length / 2)];

async function measure(browser, c, url) {
  const ctx = await browser.newContext({ locale: 'en-US', viewport: { width: 1400, height: 900 } });
  const page = await ctx.newPage();
  let requests = 0;
  let bytes = 0;
  page.on('request', () => requests++);
  page.on('requestfinished', async (r) => {
    const s = await r.sizes().catch(() => null);
    if (s) bytes += s.responseBodySize + s.responseHeadersSize;
  });

  await page.goto(url, { waitUntil: 'load' });
  await page.waitForTimeout(300);
  const loginPage = { bytes, requests };

  let t = Date.now();
  await c.login(page);
  await c.inbox(page).waitFor({ timeout: 30000 });
  const loginToInboxMs = Date.now() - t;

  await c.beforeFolder?.(page);
  t = Date.now();
  await c.openFolder(page);
  await c.folderRow(page).waitFor({ timeout: 60000 });
  const openFolderMs = Date.now() - t;

  t = Date.now();
  await c.openMessage(page);
  await c.messageShown(page).waitFor({ timeout: 30000 });
  const openMessageMs = Date.now() - t;

  await page.waitForTimeout(500);
  const result = { loginPage, loginToInboxMs, openFolderMs, openMessageMs, sessionBytes: bytes, sessionRequests: requests };
  await ctx.close();
  return result;
}

const browser = await chromium.launch();
const out = {};
for (const [name, c] of Object.entries(clients)) {
  if (!args[name]) continue;
  const runs = [];
  for (let i = 0; i < RUNS; i++) runs.push(await measure(browser, c, args[name]));
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

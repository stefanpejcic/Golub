// SPDX-License-Identifier: GPL-3.0-or-later
// Captures the wiki screenshots from the same invented demo mailbox as the README ones.
// Usage: GOLUB_URL=http://localhost:8080 E2E_PASSWORD=... node screenshots/docs.mjs <output dir>
import { chromium, devices } from '@playwright/test';
import { mkdirSync } from 'node:fs';
import { base, login, seed, user } from './demo.mjs';

const out = process.argv[2];
if (!out) throw new Error('usage: node screenshots/docs.mjs <output dir>');
mkdirSync(out, { recursive: true });

const desktop = { viewport: { width: 1440, height: 900 }, deviceScaleFactor: 1, locale: 'en-US', timezoneId: 'Europe/Belgrade', colorScheme: 'light' };
const PASS = 'demo passphrase';
const sections = ['general', 'mailbox', 'display', 'compose', 'special', 'folders', 'identities', 'responses', 'pgp', 'about', 'shortcuts'];

const shot = async (page, name) => {
  // the pointer would leave hover states on whatever it last touched
  await page.mouse.move(1435, 895);
  await page.waitForTimeout(500);
  await page.screenshot({ path: `${out}/${name}.png` });
  console.log(name);
};
const row = (page, subject) => page.getByTestId('message').filter({ hasText: subject });

await seed();
const browser = await chromium.launch();

{
  const ctx = await browser.newContext(desktop);
  const page = await ctx.newPage();
  await page.goto(base);
  await page.locator('input[type=email]').fill(user);
  await shot(page, 'login');

  await login(page);
  await shot(page, 'mailbox');

  await row(page, 'Design review moved').getByRole('link').click();
  await page.getByTestId('message-subject').waitFor();
  await shot(page, 'message');
  await page.getByTestId('action-more').click();
  await page.getByTestId('action-more-menu').waitFor();
  await shot(page, 'message-more');
  await page.keyboard.press('Escape');

  await row(page, 'Your trip to Lisbon').getByRole('link').click();
  await page.getByTestId('images-blocked').waitFor();
  await shot(page, 'message-images');

  // from the bare list, so the open message does not count as a third selected one
  await page.goto(`${base}/mail/INBOX`);
  await row(page, 'Quarterly numbers').getByRole('link').click({ modifiers: ['ControlOrMeta'] });
  await row(page, 'Invoice 2026-118').getByRole('link').click({ modifiers: ['ControlOrMeta'] });
  await page.getByTestId('bulk-bar').waitFor();
  await shot(page, 'list-select');
  await page.keyboard.press('Escape');

  await page.getByTestId('list-options').click();
  await page.getByTestId('options-threads').waitFor();
  await shot(page, 'list-options');
  await page.getByTestId('options-threads').check();
  await page.getByTestId('options-save').click();
  await page.getByTestId('thread-count').first().waitFor();
  await page.getByTestId('thread-toggle').first().click();
  await shot(page, 'threads');
  await page.getByTestId('list-options').click();
  await page.getByTestId('options-threads').uncheck();
  await page.getByTestId('options-save').click();

  await page.getByTestId('search').fill('invoice');
  await page.getByTestId('search').press('Enter');
  await page.getByTestId('search-filters').click();
  await page.getByTestId('filter-apply').waitFor();
  await shot(page, 'search');
  await page.keyboard.press('Escape');
  await page.getByTestId('search').fill('');
  await page.getByTestId('search').press('Enter');

  await page.getByTestId('compose-open').click();
  await page.getByTestId('compose-to-input').fill('ana@golub.test');
  await page.getByTestId('compose-to-input').press('Enter');
  await page.getByTestId('compose-subject').fill('Mockups look great');
  await page.getByTestId('compose-editor').locator('p').first().click();
  await page.keyboard.type('Hi Ana, thanks for the mockups. Thursday at 10:00 works for me.');
  await shot(page, 'compose');
  await page.getByTestId('compose-discard').click();
  await page.getByTestId('dialog-confirm').click().catch(() => undefined);

  await page.goto(`${base}/contacts`);
  await page.getByTestId('contact').first().click();
  await shot(page, 'contacts');

  await page.setViewportSize({ width: 1440, height: 1000 });
  for (const s of sections) {
    await page.goto(`${base}/settings/${s}`);
    await page.getByTestId('settings-title').waitFor();
    await shot(page, `settings-${s}`);
  }
  await ctx.close();
}

{
  const ctx = await browser.newContext({ ...desktop, colorScheme: 'dark' });
  const page = await ctx.newPage();
  await login(page);
  await row(page, 'Design review moved').getByRole('link').click();
  await page.getByTestId('message-subject').waitFor();
  await shot(page, 'dark');
  await ctx.close();
}

{
  const ctx = await browser.newContext({ ...desktop, locale: 'ar' });
  const page = await ctx.newPage();
  await page.goto(base);
  await page.locator('input[type=email]').fill(user);
  await shot(page, 'arabic-login');
  await ctx.close();
}

{
  const { defaultBrowserType: _, ...iphone } = devices['iPhone 13'];
  const ctx = await browser.newContext({ ...iphone, locale: 'en-US', timezoneId: 'Europe/Belgrade', colorScheme: 'light' });
  const page = await ctx.newPage();
  await login(page);
  await page.waitForTimeout(500);
  await page.screenshot({ path: `${out}/mobile-list.png` });
  await page.getByTestId('drawer-toggle').tap();
  await page.waitForTimeout(500);
  await page.screenshot({ path: `${out}/mobile-menu.png` });
  await page.keyboard.press('Escape');
  await row(page, 'Design review moved').getByRole('link').tap();
  await page.getByTestId('message-subject').waitFor();
  await page.waitForTimeout(800);
  await page.evaluate(() => document.querySelectorAll('*').forEach((el) => (el.scrollTop = 0)));
  await page.screenshot({ path: `${out}/mobile-message.png` });
  console.log('mobile');
  await ctx.close();
}

// last, because it sends a message that would show up in the other screenshots
{
  const ctx = await browser.newContext({ ...desktop, viewport: { width: 1440, height: 1000 } });
  const page = await ctx.newPage();
  await login(page);
  await page.goto(`${base}/settings/pgp`);
  await page.getByTestId('pgp-pass1').fill(PASS);
  await page.getByTestId('pgp-pass2').fill(PASS);
  await page.getByTestId('pgp-create').click();
  await page.getByTestId('pgp-own-key').first().waitFor();
  await shot(page, 'settings-pgp-keys');

  await page.getByTestId('settings-back').click();
  await page.getByTestId('compose-open').click();
  await page.getByTestId('compose-to-input').fill(user);
  await page.getByTestId('compose-to-input').press('Enter');
  await page.getByTestId('compose-subject').fill('Contract draft');
  await page.getByTestId('compose-editor').locator('p').first().click();
  await page.keyboard.type('The contract draft is below. Only you and I can read this message.');
  await page.getByTestId('compose-encrypt').check();
  await page.getByTestId('compose-sign').check();
  await shot(page, 'compose-pgp');
  await page.getByTestId('compose-send').click();
  await page.getByTestId('dialog-input').fill(PASS);
  await shot(page, 'pgp-unlock');
  await page.getByTestId('dialog-confirm').click();
  await page.getByTestId('toast').filter({ hasText: 'sent' }).waitFor({ timeout: 30_000 });

  await page.locator('[data-testid=folder][data-folder=Sent]').click();
  await row(page, 'Contract draft').getByRole('link').click();
  await page.getByTestId('pgp-text').filter({ hasText: 'contract draft' }).waitFor({ timeout: 30_000 });
  await shot(page, 'pgp-message');
  await ctx.close();
}

await browser.close();
console.log('screenshots written to', out);

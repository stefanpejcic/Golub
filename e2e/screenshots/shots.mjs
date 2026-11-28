// SPDX-License-Identifier: GPL-3.0-or-later
// Fills bob's test mailbox with made-up mail and captures the README screenshots.
// Usage: GOLUB_URL=http://localhost:8080 E2E_PASSWORD=... node screenshots/shots.mjs
import { chromium, devices } from '@playwright/test';
import { mkdirSync } from 'node:fs';
import { base, login, seed, user } from './demo.mjs';

const out = new URL('../../assets/screenshots/', import.meta.url).pathname;
mkdirSync(out, { recursive: true });

async function shoot() {
  const browser = await chromium.launch();
  const desktop = { viewport: { width: 1440, height: 900 }, deviceScaleFactor: 1, locale: 'en-US', timezoneId: 'Europe/Belgrade' };

  for (const scheme of ['light', 'dark']) {
    const ctx = await browser.newContext({ ...desktop, colorScheme: scheme });
    const page = await ctx.newPage();
    await login(page);
    await page.getByTestId('message').filter({ hasText: 'Design review moved' }).getByRole('link').click();
    await page.getByTestId('message-subject').waitFor();
    await page.mouse.move(1000, 600);
    await page.waitForTimeout(800);
    await page.screenshot({ path: `${out}/inbox-${scheme}.png` });
    if (scheme === 'light') {
      await page.getByTestId('compose-open').click();
      await page.getByTestId('compose-to-input').pressSequentially('team', { delay: 40 });
      await page.waitForTimeout(400);
      await page.screenshot({ path: `${out}/compose.png` });
      await page.keyboard.press('Escape');
      await page.goto(`${base}/contacts`);
      await page.getByTestId('contact').first().click();
      await page.waitForTimeout(400);
      await page.screenshot({ path: `${out}/contacts.png` });
    }
    await ctx.close();
  }

  const login0 = await browser.newContext({ ...desktop, colorScheme: 'light' });
  const lp = await login0.newPage();
  await lp.goto(base);
  await lp.locator('input[type=email]').fill(user);
  await lp.waitForTimeout(400);
  await lp.screenshot({ path: `${out}/login.png` });
  await login0.close();

  const { defaultBrowserType: _, ...iphone } = devices['iPhone 13'];
  const phone = await browser.newContext({ ...iphone, locale: 'en-US', timezoneId: 'Europe/Belgrade', colorScheme: 'light' });
  const mp = await phone.newPage();
  await login(mp);
  await mp.waitForTimeout(500);
  await mp.screenshot({ path: `${out}/mobile-list.png` });
  await mp.getByTestId('message').filter({ hasText: 'Design review moved' }).getByRole('link').tap();
  await mp.getByTestId('message-subject').waitFor();
  await mp.waitForTimeout(800);
  await mp.evaluate(() => document.querySelectorAll('*').forEach((el) => (el.scrollTop = 0)));
  await mp.screenshot({ path: `${out}/mobile-message.png` });
  await social(browser);
  await browser.close();
}

// GitHub's social preview: 1280x640, mascot and name on the left, the app on the right.
async function social(browser) {
  const { readFileSync } = await import('node:fs');
  const asset = (f, type) => `data:${type};base64,${readFileSync(new URL(f, new URL('../../assets/', import.meta.url))).toString('base64')}`;
  const page = await browser.newPage({ viewport: { width: 1280, height: 640 } });
  await page.setContent(`<!doctype html><html><body style="margin:0">
    <div style="width:1280px;height:640px;display:flex;align-items:center;overflow:hidden;font-family:Inter,system-ui,sans-serif;
                background:radial-gradient(circle at 20% 30%,#e0f7fa,#f8fafc 60%)">
      <div style="flex:none;width:470px;padding-left:72px">
        <img src="${asset('golub-mascot.svg', 'image/svg+xml')}" width="120" height="120">
        <div style="font-size:76px;font-weight:800;letter-spacing:-2px;color:#0d0d0d;margin-top:8px">Golub</div>
        <div style="font-size:30px;line-height:1.3;color:#334155;margin-top:10px">Privacy-first webmail<br>in a single Go binary</div>
        <div style="font-size:20px;color:#64748b;margin-top:22px">No database · No tracking · Self-hosted</div>
      </div>
      <img src="${asset('screenshots/inbox-light.png', 'image/png')}"
           style="width:980px;margin-left:10px;border-radius:14px;box-shadow:0 20px 60px rgba(15,23,42,.25);border:1px solid #e2e8f0">
    </div></body></html>`);
  await page.waitForTimeout(300);
  await page.screenshot({ path: `${out}/../social-preview.png` });
  await page.close();
}

await seed();
await shoot();
console.log('screenshots written to', out);

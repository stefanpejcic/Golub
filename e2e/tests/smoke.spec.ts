// SPDX-License-Identifier: GPL-3.0-or-later

import { expect, test } from '@playwright/test';
import { ALICE, PASSWORD } from './env';

const titles: Record<string, string> = {
  en: 'Sign in to your mailbox',
  'sr-Latn': 'Prijavite se na poštansko sanduče',
};

for (const theme of ['light', 'dark'] as const) {
  for (const lang of ['en', 'sr-Latn'] as const) {
    test(`login page renders in ${theme} / ${lang}`, async ({ page, context, baseURL }) => {
      await context.addCookies([
        { name: 'golub_theme', value: theme, url: baseURL! },
        { name: 'golub_lang', value: lang, url: baseURL! },
      ]);
      const errors: string[] = [];
      page.on('console', (m) => m.type() === 'error' && !m.text().includes('401') && errors.push(m.text()));
      page.on('pageerror', (e) => errors.push(e.message));

      await page.goto('/');
      await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
      await expect(page.locator('html')).toHaveAttribute('lang', lang);
      await expect(page.getByRole('heading')).toHaveText(titles[lang]);

      const bg = await page.evaluate(() => getComputedStyle(document.body).backgroundColor);
      expect(bg).toBe(theme === 'dark' ? 'rgb(23, 23, 23)' : 'rgb(255, 255, 255)');
      expect(errors).toEqual([]);
    });
  }
}


test('theme toggle survives reload and a new tab', async ({ browser, baseURL }) => {
  const ctx = await browser.newContext({ colorScheme: 'dark', locale: 'en-US' });
  const page = await ctx.newPage();
  await page.goto(baseURL!);
  await page.getByLabel('Email address').fill(ALICE);
  await page.getByLabel('Password').fill(PASSWORD);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page.getByTestId('user-email')).toHaveText(ALICE);
  await page.locator('[data-testid=theme-select] [data-value=light]').click();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
  await page.reload();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
  const second = await ctx.newPage();
  await second.goto(baseURL!);
  await expect(second.locator('html')).toHaveAttribute('data-theme', 'light');
  await page.locator('[data-testid=theme-select] [data-value=system]').click();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'system');
  await page.reload();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'system');
  await ctx.close();
});

test('search engines are kept out', async ({ request }) => {
  const robots = await request.get('/robots.txt');
  expect(await robots.text()).toContain('Disallow: /');
  const page = await request.get('/');
  expect(page.headers()['x-robots-tag']).toBe('noindex, nofollow, noarchive');
  expect(await page.text()).toContain('<meta name="robots" content="noindex, nofollow"');
});

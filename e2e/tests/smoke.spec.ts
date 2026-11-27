// SPDX-License-Identifier: GPL-3.0-or-later

import { expect, test } from '@playwright/test';

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
      expect(bg).toBe(theme === 'dark' ? 'rgb(3, 7, 18)' : 'rgb(255, 255, 255)');
      expect(errors).toEqual([]);
    });
  }
}

test('about dialog shows version, license and source link', async ({ page }) => {
  await page.goto('/');
  await page.getByTestId('about-open').click();
  const about = page.getByTestId('about');
  await expect(about).toBeVisible();
  await expect(page.getByTestId('about-version')).toHaveText(/^(Version|Verzija) \S+/);
  await expect(about.getByRole('link', { name: /Source code|Izvorni kod/ })).toHaveAttribute('href', 'https://github.com/stefanpejcic/Golub');
  await page.getByTestId('about-close').click();
  await expect(about).toBeHidden();
});

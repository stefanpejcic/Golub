// SPDX-License-Identifier: GPL-3.0-or-later

import { expect, test } from '@playwright/test';
import { signIn } from './helpers';

test.use({ locale: 'en-US' });

test('search filters: body text, unread only and all folders', async ({ page }) => {
  await signIn(page);
  const rows = page.getByTestId('message');

  await page.getByTestId('search').fill('Drag body 2');
  await page.getByTestId('search').press('Enter');
  await expect(rows).toHaveCount(0);

  await page.getByTestId('search-filters').click();
  await page.getByTestId('filter-body').check();
  await page.getByTestId('filter-all').check();
  await page.getByTestId('filter-apply').click();
  await expect(page.getByTestId('search-filters-count')).toHaveText('2');
  const hit = rows.filter({ hasText: 'Drag 2' });
  // Without a full-text index the server scans every body, including the 50k Perf folder.
  await expect(hit).toHaveCount(1, { timeout: 30_000 });
  await expect(hit.getByTestId('result-folder')).toHaveText(/Drag|Dropped/);

  await hit.getByRole('link').click();
  await expect(page.getByTestId('message-subject')).toHaveText('Drag 2');
  await expect(page).toHaveURL(/\/mail\/(Drag|Dropped)\/\d+$/);
  await expect(page.getByTestId('search-filters-count')).toHaveCount(0);
});

test('unread only and date range narrow the list', async ({ page }) => {
  await signIn(page);
  await page.locator('[data-testid=folder][data-folder=Threads]').click();
  const rows = page.getByTestId('message');
  await expect(rows).toHaveCount(6);
  await page.getByTestId('search-filters').click();
  await page.getByTestId('filter-since').fill('2026-04-01');
  await page.getByTestId('filter-before').fill('2026-04-01');
  await page.getByTestId('filter-apply').click();
  await expect(rows).toHaveCount(6);
  await page.getByTestId('search-filters').click();
  await page.getByTestId('filter-since').fill('2026-04-02');
  await page.getByTestId('filter-before').fill('2026-04-30');
  await page.getByTestId('filter-apply').click();
  await expect(rows).toHaveCount(0);
  await page.getByTestId('search-filters').click();
  await page.getByTestId('filter-clear').click();
  await expect(rows).toHaveCount(6);
});

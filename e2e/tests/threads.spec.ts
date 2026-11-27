// SPDX-License-Identifier: GPL-3.0-or-later

import { expect, test } from '@playwright/test';
import { signIn } from './helpers';

test.use({ locale: 'en-US' });

async function setThreads(page: import('@playwright/test').Page, on: boolean) {
  await page.getByTestId('list-options').click();
  const box = page.getByTestId('options-threads');
  if ((await box.isChecked()) !== on) await box.click();
  await page.getByTestId('options-save').click();
}

test('conversations group replies under their thread, newest activity first', async ({ page }) => {
  await signIn(page);
  await page.locator('[data-testid=folder][data-folder=Threads]').click();
  await expect(page.getByTestId('folder-title')).toHaveText('Threads');
  await setThreads(page, true);
  try {
    const rows = page.getByTestId('message');
    // Budget got the last reply, then Trip plan, then the lone message.
    await expect(rows).toHaveCount(3);
    await expect(rows.nth(0)).toContainText('Budget');
    await expect(rows.nth(1)).toContainText('Trip plan');
    await expect(rows.nth(2)).toContainText('Lone message');
    await expect(page.getByTestId('range')).toContainText('of 3');

    const trip = rows.nth(1);
    await expect(trip.getByTestId('thread-count')).toHaveText('3');
    await trip.getByTestId('thread-toggle').click();
    await expect(rows).toHaveCount(5);
    await expect(rows.nth(2)).toHaveClass(/reply/);
    await expect(rows.nth(3)).toHaveClass(/reply/);
    await trip.getByTestId('thread-toggle').click();
    await expect(rows).toHaveCount(3);
  } finally {
    await setThreads(page, false);
  }
  await expect(page.getByTestId('message')).toHaveCount(6);
});

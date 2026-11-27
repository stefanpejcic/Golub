// SPDX-License-Identifier: GPL-3.0-or-later

import { expect, test } from '@playwright/test';
import { signIn } from './helpers';

test.use({ locale: 'en-US' });

test('a folder imports an mbox and exports it again', async ({ page }) => {
  const name = `Import${Date.now()}`;
  await signIn(page);
  await page.getByTestId('settings-open').click();
  await page.locator('[data-testid=settings-nav][data-section=folders]').click();
  await page.getByTestId('folder-new').fill(name);
  await page.getByTestId('folder-create').click();
  const row = page.locator(`[data-testid=folder-row][data-folder=${name}]`);
  await expect(row).toBeVisible();

  const mbox =
    'From bob@golub.test Sat Jan  3 01:05:34 2026\nFrom: Bob <bob@golub.test>\nSubject: Imported one\nDate: Sat, 3 Jan 2026 01:05:34 +0000\n\nhello\n\n' +
    'From bob@golub.test Sun Jan  4 01:05:34 2026\nFrom: Bob <bob@golub.test>\nSubject: Imported two\nDate: Sun, 4 Jan 2026 01:05:34 +0000\n\nbye\n';
  await row.getByTestId('folder-import').click();
  await page.getByTestId('folder-import-file').setInputFiles({ name: 'old.mbox', mimeType: 'application/mbox', buffer: Buffer.from(mbox) });
  await expect(page.getByTestId('toast')).toHaveText('2 messages imported.');

  const [download] = await Promise.all([page.waitForEvent('download'), row.getByTestId('folder-export').click()]);
  expect(download.suggestedFilename()).toBe(`${name}.mbox`);

  await page.getByTestId('settings-back').click();
  await page.locator(`[data-testid=folder][data-folder=${name}]`).click();
  await expect(page.getByTestId('message')).toHaveCount(2);
  await expect(page.getByTestId('message').first()).toContainText('Imported two');

  await page.getByTestId('settings-open').click();
  await page.locator('[data-testid=settings-nav][data-section=folders]').click();
  await page.locator(`[data-testid=folder-row][data-folder=${name}]`).getByTestId('folder-delete').click();
  await page.getByTestId('dialog-confirm').click();
  await expect(page.locator(`[data-testid=folder-row][data-folder=${name}]`)).toHaveCount(0);
});

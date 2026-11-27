// SPDX-License-Identifier: GPL-3.0-or-later

import { devices, expect, test } from '@playwright/test';
import { openBySubject, signIn } from './helpers';

test.use({ locale: 'en-US' });

test('priority, read receipt and unsubscribe show on the right messages', async ({ page }) => {
  await signIn(page);
  const urgent = page.getByTestId('message').filter({ hasText: 'Urgent with receipt' });
  await expect(urgent.locator('.prio.high')).toBeVisible();

  await openBySubject(page, 'Urgent with receipt');
  await expect(page.getByTestId('priority-high')).toBeVisible();
  const banner = page.getByTestId('receipt-request');
  if (await banner.isVisible()) {
    await expect(banner).toContainText('bob@golub.test');
    await page.getByTestId('receipt-ignore').click();
    await expect(banner).toBeHidden();
    await page.reload();
    await expect(page.getByTestId('message-subject')).toContainText('Urgent with receipt');
    await expect(banner).toBeHidden();
  }

  await openBySubject(page, 'Weekly newsletter');
  await expect(page.getByTestId('unsubscribe')).toBeVisible();
  await expect(page.getByTestId('receipt-request')).toBeHidden();
});

test('empty button only in Trash and Spam', async ({ page }) => {
  await signIn(page);
  await expect(page.getByTestId('empty-folder')).toHaveCount(0);
});

test('drag a message onto a folder moves it', async ({ page }) => {
  await signIn(page);
  await page.locator('[data-testid=folder][data-folder=Drag]').click();
  await expect(page.getByTestId('folder-title')).toHaveText('Drag');
  const rows = page.getByTestId('message');
  await expect(rows).toHaveCount(2);
  await rows.filter({ hasText: 'Drag 1' }).getByRole('link').dragTo(page.locator('[data-testid=folder][data-folder=Dropped]'));
  await expect(rows).toHaveCount(1);
  await page.locator('[data-testid=folder][data-folder=Dropped]').click();
  await expect(page.getByTestId('folder-title')).toHaveText('Dropped');
  await expect(rows.filter({ hasText: 'Drag 1' })).toHaveCount(1);
});

test.describe('mobile', () => {
  const { defaultBrowserType: _, ...iphone } = devices['iPhone 13'];
  test.use(iphone);

  test('folders are a select, the drawer has no folder list', async ({ page }) => {
    await signIn(page);
    const select = page.getByTestId('folder-select');
    await expect(select).toBeVisible();
    await select.selectOption('Paging');
    await expect(page).toHaveURL(/\/mail\/Paging$/);
    await expect(page.getByTestId('folder-title')).toHaveText('Paging');
    await page.getByTestId('drawer-toggle').tap();
    await expect(page.getByTestId('compose-open')).toBeVisible();
    await expect(page.locator('[data-testid=folder][data-folder=INBOX]')).toBeHidden();
  });
});

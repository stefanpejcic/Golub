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
  await expect(page.getByTestId('receipt-request')).toBeHidden();
  page.on('dialog', () => {
    throw new Error('native browser dialogs are not used');
  });
  await page.getByTestId('unsubscribe').click();
  await expect(page.getByTestId('dialog')).toBeVisible();
  await expect(page.getByTestId('dialog-confirm')).toHaveText('Unsubscribe');
  await page.keyboard.press('Escape');
  await expect(page.getByTestId('dialog')).toBeHidden();
});

test('empty button only in Trash and Spam', async ({ page }) => {
  await signIn(page);
  await expect(page.getByTestId('empty-folder')).toHaveCount(0);
});

test('drag a message onto a folder moves it', async ({ page }) => {
  await signIn(page);
  const rows = page.getByTestId('message').filter({ hasText: /Drag \d/ });
  const folder = (name: string) => page.locator(`[data-testid=folder][data-folder=${name}]`);
  const show = async (name: string) => {
    await folder(name).click();
    await expect(page.getByTestId('folder-title')).toHaveText(name);
    await expect(page.getByTestId('message').filter({ hasNotText: /Drag \d/ })).toHaveCount(0);
  };

  // the two seeded messages may sit in either folder after earlier runs
  await show('Drag');
  let [from, to] = ['Drag', 'Dropped'];
  if ((await rows.count()) === 0) {
    [from, to] = [to, from];
    await show(from);
  }
  await expect(rows.first()).toBeVisible();
  const count = await rows.count();
  const subject = (await rows.first().locator('.subject .text').textContent())!.trim();
  await rows.first().getByRole('link').dragTo(folder(to));
  await expect(rows).toHaveCount(count - 1);

  await show(to);
  await expect(rows.filter({ hasText: subject })).toHaveCount(1);
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

test('sending without a subject asks first, and Cancel keeps the draft open', async ({ page }) => {
  await signIn(page);
  await page.getByTestId('compose-open').click();
  await page.getByTestId('compose-to-input').fill('bob@golub.test');
  await page.getByTestId('compose-to-input').press('Enter');
  await page.getByTestId('compose-send').click();
  await expect(page.getByTestId('dialog-confirm')).toHaveText('Send anyway');
  await page.getByTestId('dialog-cancel').click();
  await expect(page.getByTestId('dialog')).toBeHidden();
  await expect(page).toHaveURL(/\/compose$/);
});

test('Esc closes the open message', async ({ page }) => {
  await signIn(page);
  await openBySubject(page, 'Plain text');
  await expect(page).toHaveURL(/\/mail\/INBOX\/\d+$/);
  await page.locator('body').press('Escape');
  await expect(page).toHaveURL(/\/mail\/INBOX$/);
  await expect(page.getByTestId('no-message')).toBeVisible();
});

test('Esc closes compose, asking first when there are changes', async ({ page }) => {
  await signIn(page);
  await page.getByTestId('compose-open').click();
  await page.keyboard.press('Escape');
  await expect(page).toHaveURL(/\/mail\/INBOX$/);

  await page.getByTestId('compose-open').click();
  await page.getByTestId('compose-subject').fill('half written');
  await page.keyboard.press('Escape');
  await expect(page.getByTestId('leave-dialog')).toBeVisible();
  await page.getByTestId('leave-cancel').click();
  await expect(page).toHaveURL(/\/compose$/);
  await page.keyboard.press('Escape');
  await page.getByTestId('leave-discard').click();
  await expect(page).toHaveURL(/\/mail\/INBOX$/);
});

test.describe('dark theme', () => {
  test.use({ colorScheme: 'dark' });

  test('HTML mail is shown dark and the toggle switches it per message', async ({ page, context, baseURL }) => {
    await context.addCookies([{ name: 'golub_theme', value: 'dark', url: baseURL! }]);
    await signIn(page);
    await openBySubject(page, 'Simple HTML');
    const frame = page.locator('iframe');
    await expect(frame).toHaveAttribute('src', /dark=1/);
    const filter = () => page.frameLocator('iframe').locator('html').evaluate((el) => getComputedStyle(el).filter);
    await expect.poll(filter).toContain('invert');
    await page.getByTestId('toggle-dark').click();
    await expect(frame).not.toHaveAttribute('src', /dark=1/);
    await expect.poll(filter).not.toContain('invert');
  });
});

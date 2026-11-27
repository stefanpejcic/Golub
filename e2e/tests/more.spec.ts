// SPDX-License-Identifier: GPL-3.0-or-later

import { expect, test } from '@playwright/test';
import { openBySubject, signIn } from './helpers';

test.use({ locale: 'en-US' });

test('all headers and plain/HTML switch in the message header', async ({ page }) => {
  await signIn(page);
  await openBySubject(page, 'Multipart alternative');
  await expect(page.getByTestId('message-body')).toBeVisible();
  await page.getByTestId('toggle-plain').click();
  await expect(page.getByTestId('message-text')).toContainText('Plain part.');
  await page.getByTestId('toggle-plain').click();
  await expect(page.getByTestId('message-body')).toBeVisible();

  await page.getByTestId('toggle-headers').click();
  await expect(page.getByTestId('raw-headers')).toContainText('Message-ID: <multipart-alternative@golub.test>');
  await expect(page.getByTestId('raw-headers')).not.toContainText('Plain part.');
  await page.getByTestId('toggle-headers').click();
  await expect(page.getByTestId('raw-headers')).toHaveCount(0);
});

test('more menu: download, source, new window and print view', async ({ page, context }) => {
  await signIn(page);
  await openBySubject(page, 'Plain text');

  await page.getByTestId('action-more').click();
  const [download] = await Promise.all([page.waitForEvent('download'), page.getByTestId('more-download').click()]);
  expect(download.suggestedFilename()).toBe('Plain text.eml');

  await page.getByTestId('action-more').click();
  const [source] = await Promise.all([context.waitForEvent('page'), page.getByTestId('more-source').click()]);
  await expect(source.locator('body')).toContainText('Subject: Plain text');
  await source.close();

  await page.getByTestId('action-more').click();
  const [win] = await Promise.all([context.waitForEvent('page'), page.getByTestId('more-window').click()]);
  await expect(win.getByTestId('message-subject')).toHaveText('Plain text');
  await expect(win.getByTestId('message-text')).toContainText('čćžšđ');
  await expect(win.getByTestId('folder')).toHaveCount(0);
  await expect(win.getByTestId('back')).toHaveCount(0);
  await win.close();

  await page.getByTestId('action-more').click();
  const [printView] = await Promise.all([context.waitForEvent('page'), page.getByTestId('more-print').click()]);
  await expect(printView).toHaveURL(/print=1/);
  await expect(printView.getByTestId('message-subject')).toHaveText('Plain text');
  await printView.close();
});

test('more menu: copy keeps the original, edit as new opens the composer', async ({ page }) => {
  await signIn(page);
  await openBySubject(page, 'Inline image');
  await page.getByTestId('action-more').click();
  await page.getByTestId('action-copy').selectOption('Junk');
  await expect(page.getByTestId('toast')).toHaveText('Copied to Spam.');
  await expect(page.getByTestId('message-subject')).toHaveText('Inline image');
  await page.locator('[data-testid=folder][data-folder=Junk]').click();
  await expect(page.getByTestId('message').filter({ hasText: 'Inline image' })).toHaveCount(1);

  await page.locator('[data-testid=folder][data-folder=INBOX]').click();
  await openBySubject(page, 'Simple HTML');
  await page.getByTestId('action-more').click();
  await page.getByTestId('more-edit').click();
  await expect(page).toHaveURL(/\/compose\?mode=edit/);
  await expect(page.getByTestId('compose-subject')).toHaveValue('Simple HTML');
});

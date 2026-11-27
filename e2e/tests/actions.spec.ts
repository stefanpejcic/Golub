// SPDX-License-Identifier: GPL-3.0-or-later

import { expect, test, type Page } from '@playwright/test';
import { signIn } from './helpers';

test.use({ locale: 'en-US' });

async function openActions(page: Page, subject: string) {
  await signIn(page);
  await page.locator('[data-testid=folder][data-folder=Actions]').click();
  await page.getByTestId('message').filter({ hasText: subject }).getByRole('link').click();
  await expect(page.getByTestId('message-subject')).toHaveText(subject);
}

const row = (page: Page, subject: string) => page.getByTestId('message').filter({ hasText: subject });
const badge = (page: Page, folder: string) => page.locator(`[data-testid=folder][data-folder="${folder}"] .badge`);
const unread = async (page: Page, folder: string) =>
  (await badge(page, folder).count()) ? Number(await badge(page, folder).textContent()) : 0;

test('mark as unread and read again', async ({ page }) => {
  await openActions(page, 'Action 1');
  await expect(row(page, 'Action 1')).not.toHaveClass(/unread/);
  const before = await unread(page, 'Actions');

  await page.getByTestId('action-seen').click();
  await expect(row(page, 'Action 1')).toHaveClass(/unread/);
  await expect(badge(page, 'Actions')).toHaveText(String(before + 1));

  await row(page, 'Action 4').getByRole('link').click();
  await expect(page.getByTestId('message-subject')).toHaveText('Action 4');
  await expect(row(page, 'Action 1')).toHaveClass(/unread/);

  await row(page, 'Action 1').getByRole('link').click();
  await expect(row(page, 'Action 1')).not.toHaveClass(/unread/);
});

test('flag and unflag', async ({ page }) => {
  await openActions(page, 'Action 2');
  await page.getByTestId('action-flag').click();
  await expect(page.getByTestId('action-flag')).toHaveAttribute('aria-pressed', 'true');
  await expect(row(page, 'Action 2').getByRole('img', { name: 'Flagged' })).toBeVisible();
  await page.reload();
  await expect(row(page, 'Action 2').getByRole('img', { name: 'Flagged' })).toBeVisible();
  await page.getByTestId('action-flag').click();
  await expect(row(page, 'Action 2').getByRole('img', { name: 'Flagged' })).toHaveCount(0);
});

test('move to another folder and back', async ({ page }) => {
  await openActions(page, 'Action 3');
  await page.getByTestId('action-move').selectOption('Junk');
  await expect(page.getByTestId('toast')).toHaveText('Moved to Spam.');
  await expect(row(page, 'Action 3')).toHaveCount(0);

  await page.locator('[data-testid=folder][data-folder=Junk]').click();
  await row(page, 'Action 3').getByRole('link').click();
  await page.getByTestId('action-move').selectOption('Actions');
  await expect(page.getByTestId('toast')).toHaveText('Moved to Actions.');
  await page.locator('[data-testid=folder][data-folder=Actions]').click();
  await expect(row(page, 'Action 3')).toHaveCount(1);
});

test('delete moves to Trash, deleting in Trash is permanent, next message opens', async ({ page }) => {
  await openActions(page, 'Action 5');
  await page.getByTestId('action-delete').click();
  await expect(page.getByTestId('toast')).toHaveText('Moved to Trash.');
  await expect(row(page, 'Action 5')).toHaveCount(0);
  await expect(page.getByTestId('message-subject')).toHaveText('Action 4');

  await page.locator('[data-testid=folder][data-folder=Trash]').click();
  await row(page, 'Action 5').getByRole('link').click();
  await expect(page.getByTestId('message-subject')).toHaveText('Action 5');
  await page.keyboard.press('Delete');
  await expect(page.getByTestId('toast')).toHaveText('Deleted permanently.');
  await expect(row(page, 'Action 5')).toHaveCount(0);
});

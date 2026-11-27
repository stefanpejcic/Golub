// SPDX-License-Identifier: GPL-3.0-or-later

import { expect, test, type Page } from '@playwright/test';
import { ALICE, PASSWORD } from './env';

async function signIn(page: Page) {
  await page.goto('/');
  await page.getByLabel('Email address').fill(ALICE);
  await page.getByLabel('Password').fill(PASSWORD);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page.getByTestId('user-email')).toHaveText(ALICE);
}

test.use({ locale: 'en-US', timezoneId: 'Europe/Belgrade' });

test.beforeEach(async ({ page }) => {
  const csp: string[] = [];
  page.on('console', (m) => m.text().includes('Content Security Policy') && csp.push(m.text()));
  (page as Page & { csp: string[] }).csp = csp;
});

test.afterEach(async ({ page }) => {
  expect((page as Page & { csp: string[] }).csp).toEqual([]);
});

test('inbox lists seeded mail with decoded legacy charsets', async ({ page }) => {
  await signIn(page);
  await expect(page).toHaveURL(/\/mail\/INBOX$/);
  await expect(page.getByTestId('folder-title')).toHaveText('Inbox');

  const inbox = page.locator('[data-testid=folder][data-folder=INBOX]');
  await expect(inbox).toContainText('Inbox');
  await expect(inbox.locator('.badge')).toHaveText('18');

  const rows = page.getByTestId('message');
  await expect(rows).toHaveCount(19);
  for (const subject of ['Pozdrav šefče', 'Čestitka', 'Привет', 'Plain text']) {
    await expect(rows.filter({ hasText: subject })).toHaveCount(1);
  }
  await expect(page.getByTestId('range')).toHaveText('1–19 of 19');
  await expect(rows.filter({ hasText: 'With attachments' }).getByRole('img', { name: 'Has attachments' })).toBeVisible();
  await expect(rows.filter({ hasText: 'Plain text' })).not.toHaveClass(/unread/);
  await expect(rows.filter({ hasText: 'Simple HTML' })).toHaveClass(/unread/);
  await expect(rows.filter({ hasText: 'Simple HTML' }).getByRole('img', { name: 'Flagged' })).toBeVisible();
  await expect(rows.filter({ hasText: 'Inline image' }).getByRole('img', { name: 'Has attachments' })).toHaveCount(0);
});

test('special folders are recognised and ordered', async ({ page }) => {
  await signIn(page);
  await expect(page.locator('[data-testid=folder][data-folder=Paging]')).toBeVisible();
  const names = await page.getByTestId('folder').allTextContents();
  const labels = names.map((n) => n.replace(/\d+$/, '').trim());
  expect(labels.slice(0, 5)).toEqual(['Inbox', 'Drafts', 'Sent', 'Spam', 'Trash']);
  expect(labels).toContain('Paging');
});

test('paging is server-side, sorted by date and follows history', async ({ page }) => {
  await signIn(page);
  await page.locator('[data-testid=folder][data-folder=Paging]').click();
  await expect(page).toHaveURL(/\/mail\/Paging$/);
  await expect(page.getByTestId('range')).toHaveText('1–50 of 120');
  await expect(page.getByTestId('message').first()).toContainText('Paging message 120');
  await expect(page.getByTestId('prev')).toBeDisabled();

  await page.getByTestId('next').click();
  await expect(page).toHaveURL(/\/mail\/Paging\?page=2$/);
  await expect(page.getByTestId('range')).toHaveText('51–100 of 120');
  await expect(page.getByTestId('message').first()).toContainText('Paging message 070');

  await page.getByTestId('next').click();
  await expect(page.getByTestId('range')).toHaveText('101–120 of 120');
  await expect(page.getByTestId('message')).toHaveCount(20);
  await expect(page.getByTestId('message').last()).toContainText('Paging message 001');
  await expect(page.getByTestId('next')).toBeDisabled();

  await page.goBack();
  await expect(page.getByTestId('range')).toHaveText('51–100 of 120');
  await page.reload();
  await expect(page.getByTestId('range')).toHaveText('51–100 of 120');
});

test('folder names and counts are localized in Serbian', async ({ page, context, baseURL }) => {
  await context.addCookies([{ name: 'golub_lang', value: 'sr-Latn', url: baseURL! }]);
  await page.goto('/');
  await page.getByLabel('Adresa e-pošte').fill(ALICE);
  await page.getByLabel('Lozinka').fill(PASSWORD);
  await page.getByRole('button', { name: 'Prijava' }).click();
  await expect(page.getByTestId('folder-title')).toHaveText('Prijemno sanduče');
  await expect(page.getByLabel('18 nepročitanih poruka')).toBeVisible();
  await page.locator('[data-testid=folder][data-folder=Paging]').click();
  await expect(page.getByTestId('range')).toHaveText('1–50 od 120');
});

test('unknown folder shows a translated error', async ({ page }) => {
  await signIn(page);
  await page.goto('/mail/DoesNotExist');
  await expect(page.getByRole('alert')).toHaveText('This folder does not exist.');
});

// SPDX-License-Identifier: GPL-3.0-or-later

import { expect, test, type Page } from '@playwright/test';
import { openBySubject, signIn } from './helpers';

test.use({ locale: 'en-US' });

async function openSettings(page: Page, section: string) {
  await signIn(page);
  await page.getByTestId('settings-open').click();
  await page.locator(`[data-testid=settings-nav][data-section=${section}]`).click();
  await expect(page).toHaveURL(new RegExp(`/settings/${section}$`));
}

test('settings open from the sidebar and lead back to mail', async ({ page }) => {
  await signIn(page);
  await page.getByTestId('settings-open').click();
  await expect(page).toHaveURL(/\/settings\/general$/);
  await expect(page.getByTestId('settings-title')).toHaveText('General');
  await page.locator('[data-testid=settings-nav][data-section=about]').click();
  await expect(page.getByTestId('about-version')).toHaveText(/^Version /);
  await page.reload();
  await expect(page.getByTestId('settings-title')).toHaveText('About');
  await page.getByTestId('settings-back').click();
  await expect(page).toHaveURL(/\/mail\/INBOX$/);
  await expect(page.getByTestId('message').first()).toBeVisible();
});

test('page size preference applies to the list and is stored on the server', async ({ page }) => {
  await openSettings(page, 'mailbox');
  await page.getByTestId('pref-pageSize').selectOption('25');
  await expect(page.getByTestId('toast')).toHaveText('Saved.');
  await page.getByTestId('settings-back').click();
  await page.locator('[data-testid=folder][data-folder=Paging]').click();
  await expect(page.getByTestId('range')).toHaveText('1–25 of 120');

  await page.context().clearCookies();
  await signIn(page);
  await page.locator('[data-testid=folder][data-folder=Paging]').click();
  await expect(page.getByTestId('range')).toHaveText('1–25 of 120');

  await page.getByTestId('settings-open').click();
  await page.locator('[data-testid=settings-nav][data-section=mailbox]').click();
  await page.getByTestId('pref-pageSize').selectOption('50');
  await expect(page.getByTestId('toast')).toHaveText('Saved.');
});

test('prefer plain text shows the text part of HTML mail', async ({ page }) => {
  await openSettings(page, 'display');
  await page.getByTestId('pref-preferPlain').check();
  await expect(page.getByTestId('toast')).toHaveText('Saved.');
  await page.getByTestId('settings-back').click();
  await openBySubject(page, 'Multipart alternative');
  await expect(page.getByTestId('message-text')).toContainText('Plain part.');
  await expect(page.getByTestId('message-body')).toHaveCount(0);

  await page.getByTestId('settings-open').click();
  await page.locator('[data-testid=settings-nav][data-section=display]').click();
  await page.getByTestId('pref-preferPlain').uncheck();
  await expect(page.getByTestId('toast')).toHaveText('Saved.');
});

test('folders can be created, renamed and deleted', async ({ page }) => {
  await openSettings(page, 'folders');
  page.on('dialog', (d) => d.accept());
  await page.getByTestId('folder-new').fill('Projekti');
  await page.getByTestId('folder-create').click();
  await expect(page.getByTestId('toast')).toHaveText('Folder created.');
  const row = page.locator('[data-testid=folder-row][data-folder=Projekti]');
  await expect(row).toBeVisible();

  await row.getByTestId('folder-rename').click();
  await page.getByTestId('folder-rename-input').fill('Klijenti');
  await page.getByTestId('folder-rename-input').press('Enter');
  await expect(page.locator('[data-testid=folder-row][data-folder=Klijenti]')).toBeVisible();

  await page.locator('[data-testid=folder-row][data-folder=Klijenti]').getByTestId('folder-delete').click();
  await expect(page.getByTestId('toast')).toHaveText('Folder deleted.');
  await expect(page.locator('[data-testid=folder-row][data-folder=Klijenti]')).toHaveCount(0);
});

test('identity and response are saved', async ({ page }) => {
  await openSettings(page, 'identities');
  await page.getByTestId('identities-item').first().click();
  await page.getByTestId('identity-name').fill('Alice Tester');
  await page.getByTestId('identity-signature').fill('--\nAlice');
  await page.getByTestId('identities-save').click();
  await expect(page.getByTestId('toast')).toHaveText('Saved.');
  await page.reload();
  await expect(page.getByTestId('identities-item').first()).toContainText('Alice Tester');

  await page.locator('[data-testid=settings-nav][data-section=responses]').click();
  await page.getByTestId('responses-add').click();
  await page.getByTestId('response-name').fill('Hvala');
  await page.getByTestId('response-text').fill('Hvala na poruci!');
  await page.getByTestId('responses-save').click();
  await expect(page.getByTestId('toast')).toHaveText('Saved.');
  await page.reload();
  await expect(page.getByTestId('responses-item').filter({ hasText: 'Hvala' }).first()).toBeVisible();
});

test('invalid identity email is rejected with a clear message', async ({ page }) => {
  await openSettings(page, 'identities');
  await page.getByTestId('identities-add').click();
  await page.getByTestId('identity-email').fill('not-an-email@');
  await page.getByTestId('identities-save').click();
  await expect(page.getByTestId('identities-item')).toHaveCount(1);
});

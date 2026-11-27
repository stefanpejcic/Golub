// SPDX-License-Identifier: GPL-3.0-or-later

import { expect, test } from '@playwright/test';
import { signIn } from './helpers';

test.use({ locale: 'en-US' });

test('contacts and groups are created and suggested while composing', async ({ page }) => {
  const id = Date.now();
  const group = `Team ${id}`;
  const name = `Zorica ${id}`;
  const email = `zorica${id}@golub.test`;
  await signIn(page);
  await page.getByTestId('contacts-open').click();
  await expect(page).toHaveURL(/\/contacts$/);

  await page.getByTestId('group-new').click();
  await page.getByTestId('dialog-input').fill(group);
  await page.getByTestId('dialog-confirm').click();
  await expect(page.locator(`[data-testid=group][data-group="${group}"]`)).toBeVisible();

  await page.getByTestId('contact-new').click();
  await page.getByTestId('contact-name').fill(name);
  await page.getByTestId('contact-emails').fill(email);
  await expect(page.getByTestId('contact-group').and(page.locator(`[value="${group}"]`))).toBeChecked();
  await page.getByTestId('contact-save').click();
  await expect(page.getByTestId('toast')).toHaveText('Saved.');
  await expect(page.getByTestId('contact').filter({ hasText: name })).toBeVisible();

  await page.reload();
  await page.locator(`[data-testid=group][data-group="${group}"]`).click();
  await expect(page.getByTestId('contact')).toHaveCount(1);

  await page.getByTestId('compose-open').click();
  await page.getByTestId('compose-to-input').pressSequentially(`zorica ${id}`.slice(0, 10));
  const sg = page.getByTestId('suggestion').filter({ hasText: email });
  await expect(sg).toBeVisible();
  await page.getByTestId('compose-to-input').fill('');
  await page.getByTestId('compose-to-input').pressSequentially(String(id).slice(0, 6));
  await page.getByTestId('compose-to-input').fill(`team ${id}`);
  await page.getByTestId('compose-to-input').press('End');
  await page.getByTestId('compose-to-input').pressSequentially(' ');
  await page.getByTestId('compose-to-input').press('Backspace');
  await expect(page.getByTestId('suggestion').first()).toContainText(group);
  await page.getByTestId('compose-to-input').press('Enter');
  await expect(page.getByTestId('recipient')).toHaveText(`${name} <${email}>`);
});

test('vCard import and export', async ({ page }) => {
  const id = Date.now();
  await signIn(page);
  await page.getByTestId('contacts-open').click();
  await page.getByTestId('contacts-import-file').setInputFiles({
    name: 'people.vcf',
    mimeType: 'text/vcard',
    buffer: Buffer.from(`BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Imported ${id}\r\nEMAIL:imported${id}@golub.test\r\nEND:VCARD\r\n`),
  });
  await expect(page.getByTestId('toast')).toHaveText('Imported: 1 new, 0 updated.');
  await expect(page.getByTestId('contact').filter({ hasText: `Imported ${id}` })).toBeVisible();

  const [download] = await Promise.all([page.waitForEvent('download'), page.getByTestId('contacts-export').click()]);
  expect(download.suggestedFilename()).toBe('contacts.vcf');
});

test('sending collects new recipients for suggestions', async ({ page }) => {
  const id = Date.now();
  // A plus address of a real user, so nothing bounces back into the inbox other tests count.
  const to = `bob+c${id}@golub.test`;
  await signIn(page);
  await page.getByTestId('compose-open').click();
  await page.getByTestId('compose-to-input').fill(to);
  await page.getByTestId('compose-to-input').press('Enter');
  await page.getByTestId('compose-subject').fill('hello');
  await page.getByTestId('compose-send').click();
  await expect(page.getByTestId('toast')).toContainText('sent');

  await page.getByTestId('contacts-open').click();
  await page.getByTestId('group-collected').click();
  await expect(page.getByTestId('collected').filter({ hasText: to })).toBeVisible();
});

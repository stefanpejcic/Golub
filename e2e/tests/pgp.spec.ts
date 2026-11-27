// SPDX-License-Identifier: GPL-3.0-or-later

import { readFileSync } from 'node:fs';
import { expect, test } from '@playwright/test';
import { signIn } from './helpers';

test.use({ locale: 'en-US' });

const PASS = 'correct horse e2e';
const bobKey = readFileSync(new URL('./fixtures/bob-public.asc', import.meta.url), 'utf8');

test('encrypt and sign in the browser, then read our own copy in Sent', async ({ page }) => {
  test.setTimeout(90_000);
  const subject = `PGP ${Date.now()}`;
  await signIn(page);

  await page.goto('/settings/pgp');
  const own = page.getByTestId('pgp-own-key');
  const before = await own.count();
  await page.getByTestId('pgp-pass1').fill(PASS);
  await page.getByTestId('pgp-pass2').fill(PASS);
  await page.getByTestId('pgp-create').click();
  await expect(own).toHaveCount(before + 1);
  await page.getByTestId('pgp-paste').fill(bobKey);
  await page.getByTestId('pgp-import').click();
  await expect(page.getByTestId('pgp-other-key').filter({ hasText: 'bob@golub.test' })).not.toHaveCount(0);

  await page.getByTestId('settings-back').click();
  await page.getByTestId('compose-open').click();
  await page.getByTestId('compose-to-input').fill('bob@golub.test');
  await page.getByTestId('compose-to-input').press('Enter');
  await page.getByTestId('compose-subject').fill(subject);
  await page.getByTestId('compose-editor').locator('p').first().click();
  await page.keyboard.type('top secret body');
  await page.getByTestId('compose-encrypt').check();
  await page.getByTestId('compose-sign').check();
  await page.getByTestId('compose-send').click();
  // alice's key is locked, so signing asks for its passphrase
  await page.getByTestId('dialog-input').fill(PASS);
  await page.getByTestId('dialog-confirm').click();
  await expect(page.getByTestId('toast')).toContainText('sent', { timeout: 30_000 });

  await page.locator('[data-testid=folder][data-folder=Sent]').click();
  await page.getByTestId('message').filter({ hasText: subject }).getByRole('link').click();
  await expect(page.getByTestId('pgp-encrypted')).toBeVisible();
  await expect(page.getByTestId('pgp-text')).toContainText('top secret body', { timeout: 30_000 });
  await expect(page.getByTestId('pgp-signature')).toHaveAttribute('data-state', 'verified');

  const uid = page.url().split('/').pop();
  const raw = await page.request.get(`/api/message/raw?folder=Sent&uid=${uid}`);
  const text = await raw.text();
  expect(text).toContain('multipart/encrypted');
  expect(text).not.toContain('top secret body');

  await page.goto('/settings/pgp');
  await page.getByTestId('pgp-lock').click();
  await page.goto(`/mail/Sent/${uid}`);
  await page.getByTestId('pgp-unlock').click();
  await page.getByTestId('dialog-input').fill('wrong passphrase');
  await page.getByTestId('dialog-confirm').click();
  await expect(page.getByTestId('dialog-error')).toHaveText('Wrong passphrase.');
  await page.getByTestId('dialog-input').fill(PASS);
  await page.getByTestId('dialog-confirm').click();
  await expect(page.getByTestId('pgp-text')).toContainText('top secret body', { timeout: 30_000 });
});

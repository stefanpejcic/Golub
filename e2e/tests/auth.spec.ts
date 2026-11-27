// SPDX-License-Identifier: GPL-3.0-or-later

import { expect, test } from '@playwright/test';
import { ALICE, PASSWORD, ssoToken } from './env';

test.use({ locale: 'en-US' });

test('login and logout', async ({ page }) => {
  await page.goto('/');
  await page.getByLabel('Email address').fill(ALICE);
  await page.getByLabel('Password').fill(PASSWORD);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page.getByTestId('user-email')).toHaveText(ALICE);

  await page.reload();
  await expect(page.getByTestId('user-email')).toHaveText(ALICE);

  await page.getByTestId('logout').click();
  await expect(page.getByTestId('login-form')).toBeVisible();
  await page.reload();
  await expect(page.getByTestId('login-form')).toBeVisible();
});

test('wrong password shows a translated error', async ({ page }) => {
  await page.goto('/');
  await page.getByLabel('Email address').fill(ALICE);
  await page.getByLabel('Password').fill('definitely-wrong');
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page.getByTestId('login-error')).toHaveText('Wrong email address or password.');
});

test('SSO applies lang and theme, and tokens are single use', async ({ page }) => {
  const token = ssoToken(ALICE, { lang: 'sr-Latn', theme: 'dark' });
  await page.goto(`/sso?token=${encodeURIComponent(token)}`);
  await expect(page).toHaveURL(/\/$/);
  await expect(page.getByTestId('user-email')).toHaveText(ALICE);
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
  await expect(page.locator('html')).toHaveAttribute('lang', 'sr-Latn');
  await expect(page.getByTestId('logout')).toHaveText('Odjava');

  await page.getByTestId('logout').click();
  await page.goto(`/sso?token=${encodeURIComponent(token)}`);
  await expect(page.getByTestId('login-error')).toHaveText('Link za prijavu nije važeći ili je istekao. Prijavite se ponovo.');
  await expect(page).not.toHaveURL(/error=/);
});

test('SSO rejects a token for a master login', async ({ page }) => {
  await page.goto(`/sso?token=${encodeURIComponent(ssoToken(`${ALICE}*golubmaster`))}`);
  await expect(page.getByTestId('login-form')).toBeVisible();
});

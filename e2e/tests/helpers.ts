// SPDX-License-Identifier: GPL-3.0-or-later

import { expect, type Page } from '@playwright/test';
import { ALICE, PASSWORD } from './env';

export async function signIn(page: Page) {
  await page.goto('/');
  await page.getByLabel('Email address').fill(ALICE);
  await page.getByLabel('Password').fill(PASSWORD);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page.getByTestId('user-email')).toHaveText(ALICE);
}

export async function openBySubject(page: Page, subject: string) {
  await page.getByTestId('message').filter({ hasText: subject }).first().getByRole('link').click();
  await expect(page.getByTestId('message-subject')).toHaveText(subject);
}
